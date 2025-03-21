package opcua_listener

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/config"
	"github.com/influxdata/telegraf/metric"
	opcuaclient "github.com/influxdata/telegraf/plugins/common/opcua"
	"github.com/influxdata/telegraf/plugins/common/opcua/input"
)

type subscribeClientConfig struct {
	input.InputClientConfig
	SubscriptionInterval config.Duration `toml:"subscription_interval"`
	ConnectFailBehavior  string          `toml:"connect_fail_behavior"`
}

type subscribeClient struct {
	*input.OpcUAInputClient
	Config subscribeClientConfig

	sub                *opcua.Subscription
	monitoredItemsReqs []*ua.MonitoredItemCreateRequest
	eventItemsReqs     []*ua.MonitoredItemCreateRequest
	dataNotifications  chan *opcua.PublishNotificationData
	metrics            chan telegraf.Metric
	eventMetrics       chan telegraf.Metric

	ctx    context.Context
	cancel context.CancelFunc
}

func checkDataChangeFilterParameters(params *input.DataChangeFilter) error {
	switch {
	case params.Trigger != input.Status &&
		params.Trigger != input.StatusValue &&
		params.Trigger != input.StatusValueTimestamp:
		return fmt.Errorf("trigger '%s' not supported", params.Trigger)
	case params.DeadbandType != input.Absolute &&
		params.DeadbandType != input.Percent:
		return fmt.Errorf("deadband_type '%s' not supported", params.DeadbandType)
	case params.DeadbandValue == nil:
		return errors.New("deadband_value was not set")
	case *params.DeadbandValue < 0:
		return errors.New("negative deadband_value not supported")
	default:
		return nil
	}
}

func assignConfigValuesToRequest(req *ua.MonitoredItemCreateRequest, monParams *input.MonitoringParameters) error {
	req.RequestedParameters.SamplingInterval = float64(time.Duration(monParams.SamplingInterval) / time.Millisecond)

	if monParams.QueueSize != nil {
		req.RequestedParameters.QueueSize = *monParams.QueueSize
	}

	if monParams.DiscardOldest != nil {
		req.RequestedParameters.DiscardOldest = *monParams.DiscardOldest
	}

	if monParams.DataChangeFilter != nil {
		if err := checkDataChangeFilterParameters(monParams.DataChangeFilter); err != nil {
			return fmt.Errorf(err.Error()+", node '%s'", req.ItemToMonitor.NodeID)
		}

		req.RequestedParameters.Filter = ua.NewExtensionObject(
			&ua.DataChangeFilter{
				Trigger:       ua.DataChangeTriggerFromString(string(monParams.DataChangeFilter.Trigger)),
				DeadbandType:  uint32(ua.DeadbandTypeFromString(string(monParams.DataChangeFilter.DeadbandType))),
				DeadbandValue: *monParams.DataChangeFilter.DeadbandValue,
			},
		)
	}

	return nil
}

func (sc *subscribeClientConfig) createSubscribeClient(log telegraf.Logger) (*subscribeClient, error) {
	client, err := sc.InputClientConfig.CreateInputClient(log)
	if err != nil {
		return nil, err
	}

	if err := client.InitNodeIDs(); err != nil {
		return nil, err
	}

	if err := client.InitEventNodeIDs(); err != nil {
		return nil, err
	}

	processingCtx, processingCancel := context.WithCancel(context.Background())

	subClient := &subscribeClient{
		OpcUAInputClient:   client,
		Config:             *sc,
		monitoredItemsReqs: make([]*ua.MonitoredItemCreateRequest, len(client.NodeIDs)),
		eventItemsReqs:     make([]*ua.MonitoredItemCreateRequest, len(client.EventNodeMetricMapping)),
		// 100 was chosen to make sure that the channels will not block when multiple changes come in at the same time.
		// The channel size should be increased if reports come in on Telegraf blocking when many changes come in at
		// the same time. It could be made dependent on the number of nodes subscribed to and the subscription interval.
		dataNotifications: make(chan *opcua.PublishNotificationData, 100),
		metrics:           make(chan telegraf.Metric, 100),
		ctx:               processingCtx,
		cancel:            processingCancel,
	}

	log.Debugf("Creating monitored items")
	for i, nodeID := range client.NodeIDs {
		// The node id index (i) is used as the handle for the monitored item
		req := opcua.NewMonitoredItemCreateRequestWithDefaults(nodeID, ua.AttributeIDValue, uint32(i))
		if err := assignConfigValuesToRequest(req, &client.NodeMetricMapping[i].Tag.MonitoringParams); err != nil {
			return nil, err
		}
		subClient.monitoredItemsReqs[i] = req
	}

	log.Debugf("Creating event streaming items")
	client.EventClientHandle = make(map[uint32]input.EventNodeMetricMapping)
	for i, node := range client.EventNodeMetricMapping {
		req := opcua.NewMonitoredItemCreateRequestWithDefaults(node.NodeID, ua.AttributeIDValue, uint32(i))
		if node.SamplingInterval > 0 {
			req.RequestedParameters.SamplingInterval = float64(node.SamplingInterval)
		}
		filterExtObj, err := createEventFilter(node)
		if err != nil {
			return nil, fmt.Errorf("failed to create event filter: %w", err)
		}
		req.RequestedParameters.Filter = filterExtObj
		client.EventClientHandle[uint32(i)] = node
		subClient.eventItemsReqs[i] = req
	}
	return subClient, nil
}

// Creation of event filter for event streaming
func createEventFilter(node input.EventNodeMetricMapping) (*ua.ExtensionObject, error) {
	selects, err := createSelectClauses(node)
	if err != nil {
		return nil, err
	}
	wheres, err := createWhereClauses(node)
	if err != nil {
		return nil, err
	}
	return &ua.ExtensionObject{
		EncodingMask: ua.ExtensionObjectBinary,
		TypeID:       &ua.ExpandedNodeID{NodeID: ua.NewNumericNodeID(0, id.EventFilter_Encoding_DefaultBinary)},
		Value: ua.EventFilter{
			SelectClauses: selects,
			WhereClause:   wheres,
		},
	}, nil
}

func createSelectClauses(node input.EventNodeMetricMapping) ([]*ua.SimpleAttributeOperand, error) {
	selects := make([]*ua.SimpleAttributeOperand, len(node.Fields))
	for i, name := range node.Fields {
		typeDefinition, err := determineNodeIDType(node)
		if err != nil {
			return nil, err
		}
		selects[i] = &ua.SimpleAttributeOperand{
			TypeDefinitionID: typeDefinition,
			BrowsePath:       []*ua.QualifiedName{{NamespaceIndex: 0, Name: name}},
			AttributeID:      ua.AttributeIDValue,
		}
	}
	return selects, nil
}

func createWhereClauses(node input.EventNodeMetricMapping) (*ua.ContentFilter, error) {
	if len(node.SourceNames) == 0 {
		return &ua.ContentFilter{
			Elements: make([]*ua.ContentFilterElement, 0),
		}, nil
	}
	operands := make([]*ua.ExtensionObject, 0)
	for _, sourceName := range node.SourceNames {
		literalOperand := &ua.ExtensionObject{
			EncodingMask: 1,
			TypeID: &ua.ExpandedNodeID{
				NodeID: ua.NewNumericNodeID(0, id.LiteralOperand_Encoding_DefaultBinary),
			},
			Value: ua.LiteralOperand{
				Value: ua.MustVariant(sourceName),
			},
		}
		operands = append(operands, literalOperand)
	}

	typeDefinition, err := determineNodeIDType(node)
	if err != nil {
		return nil, err
	}

	attributeOperand := &ua.ExtensionObject{
		EncodingMask: ua.ExtensionObjectBinary,
		TypeID: &ua.ExpandedNodeID{
			NodeID: ua.NewNumericNodeID(0, id.SimpleAttributeOperand_Encoding_DefaultBinary),
		},
		Value: &ua.SimpleAttributeOperand{
			TypeDefinitionID: typeDefinition,
			BrowsePath: []*ua.QualifiedName{
				{NamespaceIndex: 0, Name: "SourceName"},
			},
			AttributeID: ua.AttributeIDValue,
		},
	}

	filterElement := &ua.ContentFilterElement{
		FilterOperator: ua.FilterOperatorInList,
		FilterOperands: append([]*ua.ExtensionObject{attributeOperand}, operands...),
	}

	wheres := &ua.ContentFilter{
		Elements: []*ua.ContentFilterElement{filterElement},
	}

	return wheres, nil
}

func determineNodeIDType(node input.EventNodeMetricMapping) (*ua.NodeID, error) {
	switch node.EventType.Type() {
	case ua.NodeIDTypeGUID:
		return ua.NewGUIDNodeID(node.EventType.Namespace(), node.EventType.StringID()), nil
	case ua.NodeIDTypeString:
		return ua.NewStringNodeID(node.EventType.Namespace(), node.EventType.StringID()), nil
	case ua.NodeIDTypeByteString:
		return ua.NewByteStringNodeID(node.EventType.Namespace(), []byte(node.EventType.StringID())), nil
	case ua.NodeIDTypeTwoByte:
		id := node.EventType.IntID()
		if id > 255 {
			return nil, fmt.Errorf("twoByte EventType requires a value in the range 0-255, got %d", id)
		}
		return ua.NewTwoByteNodeID(uint8(node.EventType.IntID())), nil
	case ua.NodeIDTypeFourByte:
		return ua.NewFourByteNodeID(uint8(node.EventType.Namespace()), uint16(node.EventType.IntID())), nil
	case ua.NodeIDTypeNumeric:
		return ua.NewNumericNodeID(node.EventType.Namespace(), node.EventType.IntID()), nil
	default:
		return nil, fmt.Errorf("unsupported NodeID type: %v", node.EventType.String())
	}
}

func (o *subscribeClient) connect() error {
	err := o.OpcUAClient.Connect(o.ctx)
	if err != nil {
		return err
	}

	o.Log.Debugf("Creating OPC UA subscription")
	o.sub, err = o.Client.Subscribe(o.ctx, &opcua.SubscriptionParameters{
		Interval: time.Duration(o.Config.SubscriptionInterval),
	}, o.dataNotifications)
	if err != nil {
		o.Log.Error("Failed to create subscription")
		return err
	}

	o.Log.Debugf("Subscribed with subscription ID %d", o.sub.SubscriptionID)
	return nil
}

func (o *subscribeClient) stop(ctx context.Context) <-chan struct{} {
	o.Log.Debugf("Stopping OPC subscription...")
	if o.State() != opcuaclient.Connected {
		return nil
	}
	if o.sub != nil {
		if err := o.sub.Cancel(ctx); err != nil {
			o.Log.Warn("Cancelling OPC UA subscription failed with error ", err)
		}
	}
	closing := o.OpcUAInputClient.Stop(ctx)
	o.cancel()
	return closing
}

func (o *subscribeClient) startStreamValues(ctx context.Context) (<-chan telegraf.Metric, error) {
	if len(o.monitoredItemsReqs) == 0 {
		return nil, nil
	}
	// fixme: Two connection attempts are made if both values and events are streamed
	err := o.connect()
	if err != nil {
		switch o.Config.ConnectFailBehavior {
		case "retry":
			o.Log.Warnf("Failed to connect to OPC UA server %s. Will attempt to connect again at the next interval: %s", o.Config.Endpoint, err)
			return nil, nil
		case "ignore":
			o.Log.Errorf("Failed to connect to OPC UA server %s. Will not retry: %s", o.Config.Endpoint, err)
			return nil, nil
		}
		return nil, err
	}
	resp, err := o.sub.Monitor(ctx, ua.TimestampsToReturnBoth, o.monitoredItemsReqs...)
	if err != nil {
		return nil, fmt.Errorf("failed to start monitoring items: %w", err)
	}
	o.Log.Debug("Monitoring items")

	for idx, res := range resp.Results {
		if !o.StatusCodeOK(res.StatusCode) {
			// Verify NodeIDs array has been built before trying to get item; otherwise show '?' for node id
			if len(o.OpcUAInputClient.NodeIDs) > idx {
				o.Log.Debugf("Failed to create monitored item for node %v (%v)",
					o.OpcUAInputClient.NodeMetricMapping[idx].Tag.FieldName, o.OpcUAInputClient.NodeIDs[idx].String())
			} else {
				o.Log.Debugf("Failed to create monitored item for node %v (%v)", o.OpcUAInputClient.NodeMetricMapping[idx].Tag.FieldName, '?')
			}

			return nil, fmt.Errorf("creating monitored item failed with status code: %w", res.StatusCode)
		}
	}

	go o.processReceivedNotifications()

	return o.metrics, nil
}

func (o *subscribeClient) startStreamEvents(ctx context.Context) (<-chan telegraf.Metric, error) {
	if len(o.eventItemsReqs) == 0 {
		return nil, nil
	}
	// fixme: Two connection attempts are made if both values and events are streamed
	err := o.connect()
	if err != nil {
		switch o.Config.ConnectFailBehavior {
		case "retry":
			o.Log.Warnf("Failed to connect to OPC UA server %s. Will attempt to connect again at the next interval: %s", o.Config.Endpoint, err)
			return nil, nil
		case "ignore":
			o.Log.Errorf("Failed to connect to OPC UA server %s. Will not retry: %s", o.Config.Endpoint, err)
			return nil, nil
		}
		return nil, err
	}
	resp, err := o.sub.Monitor(ctx, ua.TimestampsToReturnBoth, o.eventItemsReqs...)
	if err != nil {
		return nil, fmt.Errorf("failed to start monitoring event stream: %w", err)
	}
	o.Log.Debug("Monitoring events")

	for _, res := range resp.Results {
		if !o.StatusCodeOK(res.StatusCode) {
			return nil, fmt.Errorf("creating monitored event streaming item failed with status code: %w", res.StatusCode)
		}
	}

	o.eventMetrics = make(chan telegraf.Metric, 100)
	go o.processReceivedNotifications()

	return o.eventMetrics, nil
}

func (o *subscribeClient) processReceivedNotifications() {
	for {
		select {
		case <-o.ctx.Done():
			o.Log.Debug("Processing received notifications stopped")
			return

		case res, ok := <-o.dataNotifications:
			if !ok {
				o.Log.Debugf("Data notification channel closed. Processing of received notifications stopped")
				return
			}
			if res.Error != nil {
				o.Log.Error(res.Error)
				continue
			}
			if res.Value == nil {
				o.Log.Error("Received nil notification")
				return
			}

			switch notif := res.Value.(type) {
			case *ua.DataChangeNotification:
				o.Log.Debugf("Received data change notification with %d items", len(notif.MonitoredItems))
				// It is assumed the notifications are ordered chronologically
				for _, monitoredItemNotif := range notif.MonitoredItems {
					i := int(monitoredItemNotif.ClientHandle)
					oldValue := o.LastReceivedData[i].Value
					o.UpdateNodeValue(i, monitoredItemNotif.Value)
					o.Log.Debugf("Data change notification: node %q value changed from %v to %v",
						o.NodeIDs[i].String(), oldValue, o.LastReceivedData[i].Value)
					o.metrics <- o.MetricForNode(i)
				}
			case *ua.EventNotificationList:
				o.Log.Debugf("Processing event notification with %d events", len(notif.Events))
				for _, event := range notif.Events {
					fields := make(map[string]interface{})
					node := o.OpcUAInputClient.EventClientHandle[event.ClientHandle]
					for i, field := range event.EventFields {
						fieldName := node.Fields[i]
						value := field.Value()

						if value == nil {
							o.Log.Warnf("Field %s has nil value", fieldName)
							continue
						}

						if fieldName == "Message" {
							if localizedText, ok := value.(*ua.LocalizedText); ok {
								fields["Message"] = localizedText.Text
							} else {
								o.Log.Warnf("Message field is not of type *ua.LocalizedText: %T", value)
							}
							continue
						}
						fields[fieldName] = value
					}
					tags := map[string]string{
						"node_id": node.NodeID.String(),
						"source":  o.Config.Endpoint,
					}
					o.eventMetrics <- metric.New("opcua_event", tags, fields, time.Now())
				}
			default:
				o.Log.Warnf("Received notification has unexpected type %s", reflect.TypeOf(res.Value))
			}
		}
	}
}
