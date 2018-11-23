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
    }]
}

export interface AlarmSource {
    source: Alarm;
}
