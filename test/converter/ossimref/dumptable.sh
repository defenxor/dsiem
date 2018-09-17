#!/bin/bash

# this should be executed on OSSIM server, and the resulting TSVs used as input to
# ossimconverter -refdir command parameter

echo "select sid, kingdom, category from alienvault.alarm_taxonomy" | ossim-db > ossim_alarm_taxonomy.tsv
echo "select * from alienvault.alarm_kingdoms" | ossim-db > ossim_alarm_kingdom.tsv
echo "select * from alienvault.alarm_categories" | ossim-db > ossim_alarm_category.tsv
echo "select * from alienvault.product_type" | ossim-db > ossim_product_type.tsv
echo "select id,name from alienvault.category" | ossim-db > ossim_product_category.tsv
echo "SELECT id,cat_id,name FROM alienvault.subcategory;" | ossim-db > ossim_product_subcategory.tsv
