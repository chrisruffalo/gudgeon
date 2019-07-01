package engine

import (
	"database/sql"
	"fmt"

	log "github.com/sirupsen/logrus"
)

// info from db
type TopInfo struct {
	Desc  string
	Count uint64
}

// statements that flush query log data into various metrics collection points/tables
var _stmts = []string{
	// insert into list metrics when a list has a match, on conflict update by one
	"INSERT INTO list_metrics (Name, ShortName, Hits) SELECT MatchList, MatchListShort, 1 FROM buffer WHERE MatchListShort != '' ON CONFLICT(ShortName) DO UPDATE SET Hits = Hits + 1",
	// insert into rule metrics when a rule is matched, on conflict update by one
	"INSERT INTO rule_metrics (ListId, Rule, Hits) SELECT l.Id, b.MatchRule, 1 FROM buffer b JOIN list_metrics l ON b.MatchListShort = l.ShortName WHERE b.MatchRule != '' ON CONFLICT (ListId, Rule) DO UPDATE SET Hits = Hits + 1",
	// insert into client metrics, on conflict update by one
	"INSERT INTO client_metrics (Address, Count) SELECT Address, 1 FROM buffer WHERE true ON CONFLICT (Address) DO UPDATE SET Count = Count + 1",
	// insert into domain metrics, on conflict update by one
	"INSERT INTO domain_metrics (DomainName, Count) SELECT RequestDomain, 1 FROM buffer WHERE true ON CONFLICT (DomainName) DO UPDATE SET Count = Count + 1",
	// insert into query metrics, on conflict update by one
	"INSERT INTO query_metrics (QueryType, Count) SELECT RequestType, 1 FROM buffer WHERE true ON CONFLICT (QueryType) DO UPDATE SET Count = Count + 1",
	// insert new entries into the client_name table
	"INSERT INTO client_names (Address, ClientName) SELECT DISTINCT Address, ClientName FROM buffer WHERE ClientName != '' ON CONFLICT(Address) DO NOTHING",
	//  update the client names in client name with the longest client name in the buffer or that already is in the client name field
	"UPDATE client_names AS target SET (ClientName) = (SELECT ClientName FROM (SELECT b.ClientName, length(b.ClientName) as Length FROM buffer b WHERE b.ClientName != '' AND b.Address = target.Address UNION SELECT cn.ClientName, length(cn.ClientName) as Length FROM client_names cn WHERE cn.ClientName != '' AND cn.Address = target.Address ORDER BY Length DESC LIMIT 1)) WHERE true",
}

func (metrics *metrics) flush(tx *sql.Tx) {
	// start executing statements
	for _, s := range _stmts {
		_, err := tx.Exec(s)
		if err != nil {
			log.Errorf("Committing metric details (stmt=\"%s\") data: %s", s, err)
		}
	}

}

func (metrics *metrics) top(stmt string, limit int) []*TopInfo {
	if nil == metrics.db {
		return []*TopInfo{}
	}

	// limit only applied when greater than 0 (so 1 is fine)
	if limit > 0 {
		stmt += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := metrics.db.Query(stmt)
	if err != nil {
		log.Errorf("Querying top metrics: %s", err)
		return []*TopInfo{}
	}
	if rows == nil {
		return []*TopInfo{}
	}
	defer rows.Close()

	results := make([]*TopInfo, 0, limit)

	var info *TopInfo
	for rows.Next() {
		info = &TopInfo{}
		err = rows.Scan(&info.Desc, &info.Count)
		if err != nil {
			log.Errorf("Scanning top results: %s", err)
			continue
		}
		results = append(results, info)
	}

	return results
}

func (metrics *metrics) TopClients(limit int) []*TopInfo {
	return metrics.top("SELECT COALESCE(CASE WHEN n.ClientName != NULL AND n.ClientName = '' THEN c.Address ELSE n.ClientName END, c.Address) AS Name, Count FROM client_metrics c JOIN client_names n ON c.Address = n.Address ORDER BY Count DESC", limit)
}

func (metrics *metrics) TopDomains(limit int) []*TopInfo {
	return metrics.top("SELECT DomainName, Count FROM domain_metrics ORDER BY Count DESC", limit)
}

func (metrics *metrics) TopQueryTypes(limit int) []*TopInfo {
	return metrics.top("SELECT QueryType, Count FROM query_metrics ORDER BY Count DESC", limit)
}

func (metrics *metrics) TopLists(limit int) []*TopInfo {
	return metrics.top("SELECT Name, Hits FROM list_metrics ORDER BY Hits DESC", limit)
}

func (metrics *metrics) TopRules(limit int) []*TopInfo {
	return metrics.top("SELECT r.Rule, r.Hits FROM rule_metrics r JOIN list_metrics l ON l.Id = r.ListId ORDER BY r.Hits DESC", limit)
}
