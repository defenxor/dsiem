/* eslint-disable @typescript-eslint/naming-convention */
/*
Copyright (c) 2019 PT Defender Nusa Semesta and contributors, All rights reserved.

This file is part of Dsiem.

Dsiem is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation version 3 of the License.

Dsiem is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Dsiem. If not, see <https:www.gnu.org/licenses/>.
*/

// WARNING: THIS FILE IS NOT CURRENTLY USED APPROPRIATELY TO ENFORCE TYPE

export interface Alarm {
    id: string;
    title: string;
    status: string;
    kingdom: string;
    category: string;
    timestamp: string;
    update_time: string;
    risk: number;
    risk_class: string;
    tag: string;
    src_ips: string;
    dst_ips: string;
    networks: string;
    rules: [{
        timeout: number;
        protocol: string;
        from: string;
        to: string;
        port_from: string;
        port_to: string;
        plugin_id: number;
        stage: number;
        start_time: number;
        end_time: number;
        reliability: number;
        plugin_sid: number;
        occurrence: number;
        events_count: number;
    }];
}

export interface AlarmSource {
    source: Alarm;
}
