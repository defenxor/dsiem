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
    occurrence: number;
    events_count: number;
    src_ips: string;
    dst_ips: string;
    networks: string;
    rules: [{
        timeout: number;
        protocol: string;
        from: string;
        plugin_id: number;
        stage: number;
        start_time: number;
        reliability: number;
        plugin_sid: number;
    }]
}

export interface AlarmSource {
    source: Alarm;
}
