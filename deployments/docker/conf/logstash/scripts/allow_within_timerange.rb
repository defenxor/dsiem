# This is a Logstash Ruby filter plugin script to remove events whose timestamp 
# falls outside of a given time frame.
# 
# Usage:
# Put the following config in a Logstash filter section
#
# ruby {
#  path => "/replace/with/full/path/to/allow_within_timerange.rb"
#  script_params => {
#    "timestamp_field" => "timestamp"
#    "time_from" => "23:00"
#    "time_to" => "02:00"
# }
# 
# 'timestamp' in this case should be a field created/updated by Logstash date
# filter plugin (https://www.elastic.co/guide/en/logstash/current/plugins-filters-date.html).
#
# To manually run the test cases:
# logstash -e 'filter { ruby { path => "/path/to/allow_within_timerange.rb" } }' -t
#

def register(params)
  @timestamp_field = params["timestamp_field"]
  @time_from = params["time_from"]
  @time_to = params["time_to"]
end

def is_in_range(ts)
  t = Time.at(ts).to_time
  fr_hour, fr_min = @time_from.split(":").map(&:to_i)
  to_hour, to_min = @time_to.split(":").map(&:to_i)
  fr_time = Time.new(t.year, t.month, t.day, fr_hour, fr_min, 0)
  to_time = Time.new(t.year, t.month, t.day, to_hour, to_min, 59)
  if fr_time <= to_time
    return t.between?(fr_time, to_time)
  end
  if t >= fr_time || t <= to_time
    return true
  else
    return false
  end
end

def filter(event)
  ts = event.get(@timestamp_field).to_i
  if is_in_range(ts)
    return [event]
  else
    return []
  end
end

test "event within time range - 1" do
  parameters do
    { "timestamp_field" => "field", "time_from" => "00:00", "time_to" => "01:00" }
  end
  # event @ today 00:30:00 UTC
  n = Time.now()
  ts_ms = Time.new(n.year, n.month, n.day, 0, 30, 0).to_i
  in_event { { "field" => ts_ms } }
  expect("pass the event") do |events|
    events.size != 0
  end
end

test "event within time range - 2" do
  parameters do
    { "timestamp_field" => "field", "time_from" => "00:00", "time_to" => "01:00" }
  end
  # event @ yesterday 00:30:00 UTC
  n = Time.at(Time.now.to_i - 86400)
  ts_ms = Time.new(n.year, n.month, n.day, 0, 30, 0).to_i
  in_event { { "field" => ts_ms } }
  expect("pass the event") do |events|
    events.size != 0
  end
end

test "event within time range that crosses 00:00 - 1" do
  parameters do
    { "timestamp_field" => "field","time_from" => "22:00", "time_to" => "01:00" }
  end
  # event @ today 23:30:00 UTC
  n = Time.now()
  ts_ms = Time.new(n.year, n.month, n.day, 23, 30, 0).to_i
  in_event { { "field" => ts_ms } }
  expect("pass the event") do |events|
    events.size != 0
  end
end

test "event within time range that crosses 00:00 - 2" do
  parameters do
    { "timestamp_field" => "field", "time_from" => "22:00", "time_to" => "01:00" }
  end
  # event @ today 00:30:00 UTC
  n = Time.now()
  ts_ms = Time.new(n.year, n.month, n.day, 0, 30, 0).to_i
  in_event { { "timestamp" => ts_ms } }
  expect("pass the event") do |events|
    events.size != 0
  end
end

test "event outside of time range - 1" do
  parameters do
    { "timestamp_field" => "field", "time_from" => "01:00", "time_to" => "01:30" }
  end
  # event @ today 01:15:00 UTC
  n = Time.now()
  ts_ms = Time.new(n.year, n.month, n.day, 0, 15, 0).to_i
  in_event { { "field" => ts_ms } }
  expect("drop the event") do |events|
    events.size == 0
  end
end

test "event outside of time range - 2" do
  parameters do
    { "timestamp_field" => "field", "time_from" => "01:00", "time_to" => "01:30" }
  end
  # event @ today 02:00:00 UTC
  n = Time.now()
  ts_ms = Time.new(n.year, n.month, n.day, 2, 0, 0).to_i
  in_event { { "field" => ts_ms } }
  expect("drop the event") do |events|
    events.size == 0
  end
end

test "event outside of time range - 3" do
  parameters do
    { "timestamp_field" => "field", "time_from" => "01:00", "time_to" => "01:30" }
  end
  # event @ yesterday 02:00:00 UTC
  n = Time.at(Time.now.to_i - 86400)
  ts_ms = Time.new(n.year, n.month, n.day, 2, 0, 0).to_i
  in_event { { "field" => ts_ms } }
  expect("drop the event") do |events|
    events.size == 0
  end
end

test "event outside of time range that crosses 00:00 - 1" do
  parameters do
    { "timestamp_field" => "field", "time_from" => "22:00", "time_to" => "01:00" }
  end
  # event @ today 21:00:00 UTC
  n = Time.now()
  ts_ms = Time.new(n.year, n.month, n.day, 21, 0, 0).to_i
  in_event { { "field" => ts_ms } }
  expect("drop the event") do |events|
    events.size == 0
  end
end

test "event outside of time range that crosses 00:00 - 2" do
  parameters do
    { "timestamp_field" => "field", "time_from" => "22:00", "time_to" => "01:00" }
  end
  # event @ today 2:00:00 UTC
  n = Time.now()
  ts_ms = Time.new(n.year, n.month, n.day, 2, 0, 0).to_i
  in_event { { "field" => ts_ms } }
  expect("drop the event") do |events|
    events.size == 0
  end
end
