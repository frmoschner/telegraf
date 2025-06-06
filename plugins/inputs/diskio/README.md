# DiskIO Input Plugin

This plugin gathers metrics about disk traffic and timing.

⭐ Telegraf v0.10.0
🏷️ system
💻 all

## Global configuration options <!-- @/docs/includes/plugin_config.md -->

In addition to the plugin-specific configuration settings, plugins support
additional global and plugin configuration settings. These settings are used to
modify metrics, tags, and field or create aliases and configure ordering, etc.
See the [CONFIGURATION.md][CONFIGURATION.md] for more details.

[CONFIGURATION.md]: ../../../docs/CONFIGURATION.md#plugins

## Configuration

```toml @sample.conf
# Read metrics about disk IO by device
[[inputs.diskio]]
  ## Devices to collect stats for
  ## Wildcards are supported except for disk synonyms like '/dev/disk/by-id'.
  ## ex. devices = ["sda", "sdb", "vd*", "/dev/disk/by-id/nvme-eui.00123deadc0de123"]
  # devices = ["*"]

  ## Skip gathering of the disk's serial numbers.
  # skip_serial_number = true

  ## Device metadata tags to add on systems supporting it (Linux only)
  ## Use 'udevadm info -q property -n <device>' to get a list of properties.
  ## Note: Most, but not all, udev properties can be accessed this way. Properties
  ## that are currently inaccessible include DEVTYPE, DEVNAME, and DEVPATH.
  # device_tags = ["ID_FS_TYPE", "ID_FS_USAGE"]

  ## Using the same metadata source as device_tags, you can also customize the
  ## name of the device via templates.
  ## The 'name_templates' parameter is a list of templates to try and apply to
  ## the device. The template may contain variables in the form of '$PROPERTY' or
  ## '${PROPERTY}'. The first template which does not contain any variables not
  ## present for the device is used as the device name tag.
  ## The typical use case is for LVM volumes, to get the VG/LV name instead of
  ## the near-meaningless DM-0 name.
  # name_templates = ["$ID_FS_LABEL","$DM_VG_NAME/$DM_LV_NAME"]
```

### Docker container

To monitor the Docker engine host from within a container you will need to
mount the host's filesystem into the container and set the `HOST_PROC`
environment variable to the location of the `/proc` filesystem.  Additionally,
it is required to use privileged mode to provide access to `/dev`.

If you are using the `device_tags` or `name_templates` options, you will need
to bind mount `/run/udev` into the container.

```shell
docker run --privileged -v /:/hostfs:ro -v /run/udev:/run/udev:ro -e HOST_PROC=/hostfs/proc telegraf
```

## Metrics

- diskio
  - tags:
    - name (device name)
    - serial (device serial number)
  - fields:
    - reads (integer, counter)
    - writes (integer, counter)
    - read_bytes (integer, counter, bytes)
    - write_bytes (integer, counter, bytes)
    - read_time (integer, counter, milliseconds)
    - write_time (integer, counter, milliseconds)
    - io_time (integer, counter, milliseconds)
    - weighted_io_time (integer, counter, milliseconds)
    - iops_in_progress (integer, gauge)
    - merged_reads (integer, counter)
    - merged_writes (integer, counter)
    - io_util (float64, gauge, percent)
    - io_await (float64, gauge, milliseconds)
    - io_svctm (float64, gauge, milliseconds)

On linux these values correspond to the values in [`/proc/diskstats`][1] and
[`/sys/block/<dev>/stat`][2].

[1]: https://www.kernel.org/doc/Documentation/ABI/testing/procfs-diskstats

[2]: https://www.kernel.org/doc/Documentation/block/stat.txt

### `reads` & `writes`

These values increment when an I/O request completes.

### `read_bytes` & `write_bytes`

These values count the number of bytes read from or written to this
block device.

### `read_time` & `write_time`

These values count the number of milliseconds that I/O requests have
waited on this block device.  If there are multiple I/O requests waiting,
these values will increase at a rate greater than 1000/second; for
example, if 60 read requests wait for an average of 30 ms, the read_time
field will increase by 60*30 = 1800.

### `io_time`

This value counts the number of milliseconds during which the device has
had I/O requests queued.

### `weighted_io_time`

This value counts the number of milliseconds that I/O requests have waited
on this block device.  If there are multiple I/O requests waiting, this
value will increase as the product of the number of milliseconds times the
number of requests waiting (see `read_time` above for an example).

### `iops_in_progress`

This value counts the number of I/O requests that have been issued to
the device driver but have not yet completed.  It does not include I/O
requests that are in the queue but not yet issued to the device driver.

### `merged_reads` & `merged_writes`

Reads and writes which are adjacent to each other may be merged for
efficiency.  Thus two 4K reads may become one 8K read before it is
ultimately handed to the disk, and so it will be counted (and queued)
as only one I/O. These fields lets you know how often this was done.

### `io_await`

The average time per I/O operation (ms)

### `io_svctm`

The service time per I/O operation, excluding wait time (ms)

### `io_util`

The percentage of time the disk was active (%)

## Sample Queries

### Calculate percent IO utilization per disk and host

```sql
SELECT non_negative_derivative(last("io_time"),1ms) FROM "diskio" WHERE time > now() - 30m GROUP BY "host","name",time(60s)
```

### Calculate average queue depth

`iops_in_progress` will give you an instantaneous value. This will give you the
average between polling intervals.

```sql
SELECT non_negative_derivative(last("weighted_io_time"),1ms) from "diskio" WHERE time > now() - 30m GROUP BY "host","name",time(60s)
```

## Example Output

```text
diskio,name=sda1 merged_reads=0i,reads=2353i,writes=10i,write_bytes=2117632i,write_time=49i,io_time=1271i,weighted_io_time=1350i,read_bytes=31350272i,read_time=1303i,iops_in_progress=0i,merged_writes=0i 1578326400000000000
diskio,name=centos/var_log reads=1063077i,writes=591025i,read_bytes=139325491712i,write_bytes=144233131520i,read_time=650221i,write_time=24368817i,io_time=852490i,weighted_io_time=25037394i,iops_in_progress=1i,merged_reads=0i,merged_writes=0i 1578326400000000000
diskio,name=sda write_time=49i,io_time=1317i,weighted_io_time=1404i,reads=2495i,read_time=1357i,write_bytes=2117632i,iops_in_progress=0i,merged_reads=0i,merged_writes=0i,writes=10i,read_bytes=38956544i 1578326400000000000
```

```text
diskio,name=sda io_await:0.3317307692307692,io_svctm:0.07692307692307693,io_util:0.5329780146568954 1578326400000000000
diskio,name=sda1 io_await:0.3317307692307692,io_svctm:0.07692307692307693,io_util:0.5329780146568954 1578326400000000000
diskio,name=sda2 io_await:0.3317307692307692,io_svctm:0.07692307692307693,io_util:0.5329780146568954 1578326400000000000
```
