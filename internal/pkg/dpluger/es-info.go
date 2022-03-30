package dpluger

import (
	"encoding/json"
	"net/http"
)

func elasticsearchVersion(id string) (string, error) {
	info, err := elasticsearchInfo(id)
	if err != nil {
		return "", err
	}

	if info.Version.Distribution == "opensearch" {
		ver := info.Version.CompatVersion
		if ver == "" {
			ver = "0"
		}

		return ver, nil
	}

	return info.Version.Number, nil
}

type ElasticsearchInfo struct {
	Name        string `json:"name"`
	ClusterName string `json:"cluster_name"`
	Version     struct {
		Distribution   string `json:"distribution,omitempty"`
		Number         string `json:"number"`
		BuildHash      string `json:"build_hash"`
		BuildTimestamp string `json:"build_timestamp"`
		BuildSnapshot  bool   `json:"build_snapshot"`
		LuceneVersion  string `json:"lucene_version"`
		CompatVersion  string `json:"minimum_wire_compatibility_version"`
	} `json:"version"`
	TagLine string `json:"tagline"`
}

func elasticsearchInfo(url string) (*ElasticsearchInfo, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	var info ElasticsearchInfo
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return nil, err
	}

	return &info, nil
}
