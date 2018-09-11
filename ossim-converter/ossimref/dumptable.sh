#!/bin/bash

# run this on OSSIM VM

echo "select sid, kingdom, category from alienvault.alarm_taxonomy" | ossim-db > ossim_taxonomy.tsv
echo "select * from alienvault.alarm_kingdoms" | ossim-db > ossim_kingdoms.tsv
echo "select * from alienvault.alarm_categories" | ossim-db > ossim_categories.tsv

