import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { ElasticsearchService } from './elasticsearch.service';
import { url2obj } from './utilities';

const CONFIG_PATH = './assets/config/esconfig.json'

type configuration = {
    elasticsearch: string;
    kibana: string;
}

@Injectable({
    providedIn: 'root'
})

export class DsiemService {

    constructor(private es: ElasticsearchService, private http: HttpClient) {}

    public async init(): Promise<void> {
        try {
            const escfg = await this.loadCredential(CONFIG_PATH);
            await this.es.initClient(escfg.elasticsearch, escfg.kibana);
        } catch(err) {
            this.handleCaughtEror(err);
        }
    }

    public async initWithCredentials(username:string, password:string): Promise<void> {
        try {
            const escfg = await this.loadCredential(CONFIG_PATH)
            const { protocol, host } = url2obj(escfg.elasticsearch);
            await this.es.initClient(`${protocol}://${username}:${password}@${host}`, escfg.kibana);
        } catch(err) {
            this.handleCaughtEror(err);
        }
    }

    private loadCredential(path: string): Promise<configuration> {
        return this.http.get<configuration>(path).toPromise();
    }

    private handleCaughtEror(err: any) {
        throw err;
    }
}