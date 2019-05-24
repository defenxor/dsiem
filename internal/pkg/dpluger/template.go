// Copyright (c) 2018 PT Defender Nusa Semesta and contributors, All rights reserved.
//
// This file is part of Dsiem.
//
// Dsiem is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation version 3 of the License.
//
// Dsiem is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Dsiem. If not, see <https://www.gnu.org/licenses/>.

package dpluger

var templPluginCollect = `
###############################################################################
# Dsiem {{.P.Name}} Plugin
# Type: {{.P.Type}}{{if .IsPluginRule}}, Plugin ID: {{.P.Fields.PluginID}}{{end}}
# 
# Auto-generated by {{.Creator}} on {{.CreateDate}}
###############################################################################

filter {

# 1st step: identify the source log and clone it to another event with type => siem_events

  if {{.P.IdentifierField}} == "{{.P.IdentifierValue}}" {{.P.IdentifierFilter}} {
    clone {
      clones => [ "siem_events" ]
    }

# 2nd step: remove the source log identifier from the clone, so that the clone will not 
# go through the same pipeline as the source log. Also remove the temporary type field, 
# replacing it with metadata field that will be read by the rest of siem pipeline.

    if [type] == "siem_events" {
      mutate {
        remove_field => [ "{{.P.IdentifierField}}" , "type" ]
        add_field => { 
          "[@metadata][siem_plugin_type]" => "{{.P.Name}}"
          "[@metadata][siem_data_type]" => "normalizedEvent"
        }
      }
    }
  }
}

# 3rd step: the actual event normalization so that it matches the format that siem expect.
#
# Required fields:
#   timestamp (date), title (string), sensor (string), product (string), dst_ip (string), src_ip (string)
#
# For PluginRule type plugin, the following are also required:
#   plugin_id (integer), plugin_sid (integer)
#
# For TaxonomyRule type plugin, the following is also required:
#   category (string)
#
# Optional fields:
# These fields are optional but should be included whenever possible since they can be used in directive rules:
#   dst_port (integer), src_port (integer), protocol (string), subcategory (string)
# 
# These fields are also optional and can be used in directive rules. They should be used for custom data that 
# are not defined in standard SIEM fields.
#   custom_label1 (string), custom_data1 (string), custom_label2 (string), custom_data2 (string)
#   custom_label3 (string), custom_data3 (string)
#
# And this field is optional, and should be included if the original logs are also stored in elasticsearch. 
# This will allow direct pivoting from alarm view in the web UI to the source index.
#   src_index_pattern (string)
#
# As for other fields from source log, they will be removed by logstash plugin prune below

filter {
  if [@metadata][siem_plugin_type] == "{{.P.Name}}" {
    if {{.SIDFieldPlain}} in [ 
      {{$c := counter}}{{range $k,$v := .R.Sids }}{{if call $c}},
      {{end}}"{{$v.SIDTitle}}"{{end}}
    ] {

      translate {
        field => "{{.SIDField}}"
        destination => "[plugin_sid]"
        dictionary => { {{- range $k,$v := .R.Sids }}
          "{{$v.SIDTitle}}" => "{{$v.SID}}"{{end}}
        }
      }

      # the rest should be the same as nonCollect plugin
      date {
        match => [ "{{.P.Fields.Timestamp}}", "{{.P.Fields.TimestampFormat}}" ]
        target => [timestamp]
      }
      mutate {
        replace => {
          "title" => "{{.SIDField}}"
          "src_index_pattern" => "{{.P.Index}}"
          "sensor" => "{{.P.Fields.Sensor}}"
          "product" => "{{.P.Fields.Product}}"
          "src_ip" => "{{- .P.Fields.SrcIP -}}"
          "dst_ip" => "{{.P.Fields.DstIP -}}"
          "protocol" => "{{.P.Fields.Protocol}}"
          {{if .IsFieldActive "Category" }}"category" => "{{.P.Fields.Category}}"{{end}}
          {{if .IsFieldActive "SubCategory" }}"subcategory" => "{{.P.Fields.SubCategory}}"{{end}}
          {{if .IsFieldActive "PluginID" }}"plugin_id" => "{{.P.Fields.PluginID}}"{{end}}
          {{if .IsFieldActive "SrcPort" }}"src_port" => "{{.P.Fields.SrcPort -}}"{{end}}
          {{if .IsFieldActive "DstPort" }}"dst_port" => "{{.P.Fields.DstPort -}}"{{end}}
          {{if .IsFieldActive "CustomLabel1" }}"custom_label1" => "{{.P.Fields.CustomLabel1}}"{{end}}
          {{if .IsFieldActive "CustomLabel2" }}"custom_label2" => "{{.P.Fields.CustomLabel2}}"{{end}}
          {{if .IsFieldActive "CustomLabel3" }}"custom_label3" => "{{.P.Fields.CustomLabel3}}"{{end}}
          {{if .IsFieldActive "CustomData1" }}"custom_data1" => "{{.P.Fields.CustomData1}}"{{end}}
          {{if .IsFieldActive "CustomData2" }}"custom_data2" => "{{.P.Fields.CustomData2}}"{{end}}
          {{if .IsFieldActive "CustomData3" }}"custom_data3" => "{{.P.Fields.CustomData3}}"{{end}}
        }
      }
      {{if .IsIntegerMutationRequired}}
      mutate {
        convert => {
          {{if .IsFieldActive "PluginID" }}"plugin_id" => "integer"{{end}}
          {{if .IsFieldActive "PluginSID" }}"plugin_sid" => "integer"{{end}}
          {{if .IsFieldActive "SrcPort" }}"src_port" => "integer"{{end}}
          {{if .IsFieldActive "DstPort" }}"dst_port" => "integer"{{end}}
        }
      }
      {{end}}

      # delete fields except those included in the whitelist below
      prune {
        whitelist_names => [ "timestamp$", "@metadata", "^src_index_pattern$", "^title$", "^sensor$", "^product$",
          "^src_ip$", "^dst_ip$", "^plugin_id$", "^plugin_sid$", "^category$", "^subcategory$",
          "^src_port$", "^dst_port$", "^protocol$", "^custom_label1$", "^custom_label2$", "^custom_label3$",
          "^custom_data1$", "^custom_data2$", "^custom_data3$" ]
      }
    } else {
      # title doesnt match
      drop {}
    } # titleField
  } # siem_plugin_type
}
`

var templPluginNonCollect = `
###############################################################################
# Dsiem {{.P.Name}} Plugin
# Type: {{.P.Type}}{{if .IsPluginRule}}, Plugin ID: {{.P.Fields.PluginID}}{{end}}
# 
# Auto-generated by {{.Creator}} on {{.CreateDate}}
###############################################################################

filter {

# 1st step: identify the source log and clone it to another event with type => siem_events

  if {{.P.IdentifierField}} == "{{.P.IdentifierValue}}" {{.P.IdentifierFilter}} {
    clone {
      clones => [ "siem_events" ]
    }

# 2nd step: remove the source log identifier from the clone, so that the clone will not 
# go through the same pipeline as the source log. Also remove the temporary type field, 
# replacing it with metadata field that will be read by the rest of siem pipeline.

    if [type] == "siem_events" {
      mutate {
        remove_field => [ "{{.P.IdentifierField}}" , "type" ]
        add_field => { 
          "[@metadata][siem_plugin_type]" => "{{.P.Name}}"
          "[@metadata][siem_data_type]" => "normalizedEvent"
        }
      }
    }
  }
}

# 3rd step: the actual event normalization so that it matches the format that siem expect.
#
# Required fields:
#   timestamp (date), title (string), sensor (string), product (string), dst_ip (string), src_ip (string)
#
# For PluginRule type plugin, the following are also required:
#   plugin_id (integer), plugin_sid (integer)
#
# For TaxonomyRule type plugin, the following is also required:
#   category (string)
#
# Optional fields:
# These fields are optional but should be included whenever possible since they can be used in directive rules:
#   dst_port (integer), src_port (integer), protocol (string), subcategory (string)
# 
# These fields are also optional and can be used in directive rules. They should be used for custom data that 
# are not defined in standard SIEM fields.
#   custom_label1 (string), custom_data1 (string), custom_label2 (string), custom_data2 (string)
#   custom_label3 (string), custom_data3 (string)
#
# And this field is optional, and should be included if the original logs are also stored in elasticsearch. 
# This will allow direct pivoting from alarm view in the web UI to the source index.
#   src_index_pattern (string)
#
# As for other fields from source log, they will be removed by logstash plugin prune below

filter {
  if [@metadata][siem_plugin_type] == "{{.P.Name}}" {
    date {
      match => [ "{{.P.Fields.Timestamp}}", "{{.P.Fields.TimestampFormat}}" ]
      target => [timestamp]
    }
    mutate {
      replace => {
        "title" => "{{.P.Fields.Title}}"
        "src_index_pattern" => "{{.P.Index}}"
        "sensor" => "{{.P.Fields.Sensor}}"
        "product" => "{{.P.Fields.Product}}"
        "src_ip" => "{{- .P.Fields.SrcIP -}}"
        "dst_ip" => "{{.P.Fields.DstIP -}}"
        "protocol" => "{{.P.Fields.Protocol}}"
        {{if .IsFieldActive "Category" }}"category" => "{{.P.Fields.Category}}"{{end}}
        {{if .IsFieldActive "SubCategory" }}"subcategory" => "{{.P.Fields.SubCategory}}"{{end}}
        {{if .IsFieldActive "PluginID" }}"plugin_id" => "{{.P.Fields.PluginID}}"{{end}}
        {{if .IsFieldActive "PluginSID" }}"plugin_sid" => "{{.P.Fields.PluginSID}}"{{end}}
        {{if .IsFieldActive "SrcPort" }}"src_port" => "{{.P.Fields.SrcPort -}}"{{end}}
        {{if .IsFieldActive "DstPort" }}"dst_port" => "{{.P.Fields.DstPort -}}"{{end}}
        {{if .IsFieldActive "CustomLabel1" }}"custom_label1" => "{{.P.Fields.CustomLabel1}}"{{end}}
        {{if .IsFieldActive "CustomLabel2" }}"custom_label2" => "{{.P.Fields.CustomLabel2}}"{{end}}
        {{if .IsFieldActive "CustomLabel3" }}"custom_label3" => "{{.P.Fields.CustomLabel3}}"{{end}}
        {{if .IsFieldActive "CustomData1" }}"custom_data1" => "{{.P.Fields.CustomData1}}"{{end}}
        {{if .IsFieldActive "CustomData2" }}"custom_data2" => "{{.P.Fields.CustomData2}}"{{end}}
        {{if .IsFieldActive "CustomData3" }}"custom_data3" => "{{.P.Fields.CustomData3}}"{{end}}
      }
    }
    {{if .IsIntegerMutationRequired}}
    mutate {
      convert => {
        {{if .IsFieldActive "PluginID" }}"plugin_id" => "integer"{{end}}
        {{if .IsFieldActive "PluginSID" }}"plugin_sid" => "integer"{{end}}
        {{if .IsFieldActive "SrcPort" }}"src_port" => "integer"{{end}}
        {{if .IsFieldActive "DstPort" }}"dst_port" => "integer"{{end}}
      }
    }
    {{end}}

    # delete fields except those included in the whitelist below
    prune {
      whitelist_names => [ "timestamp$", "@metadata", "^src_index_pattern$", "^title$", "^sensor$", "^product$",
      "^src_ip$", "^dst_ip$", "^plugin_id$", "^plugin_sid$", "^category$", "^subcategory$",
      "^src_port$", "^dst_port$", "^protocol$", "^custom_label1$", "^custom_label2$", "^custom_label3$",
      "^custom_data1$", "^custom_data2$", "^custom_data3$" ]
    }
  }
}
`
