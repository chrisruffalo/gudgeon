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

func (metrics *metrics) flush(tx *sql.Tx) {
	// statements
	stmts := []string{
		"INSERT INTO list_metrics (Name, ShortName, Hits) SELECT MatchList, MatchListShort, 1 FROM buffer WHERE MatchListShort != '' ON CONFLICT(ShortName) DO UPDATE SET Hits = Hits + 1",
		"INSERT INTO rule_metrics (ListId, Rule, Hits) SELECT l.Id, b.MatchRule, 1 FROM buffer b JOIN list_metrics l ON b.MatchListShort = l.ShortName WHERE b.MatchRule != '' ON CONFLICT (ListId, Rule) DO UPDATE SET Hits = Hits + 1",
		"INSERT INTO client_metrics (Address, Count) SELECT Address, 1 FROM buffer WHERE true ON CONFLICT (Address) DO UPDATE SET Count = Count + 1",
		"INSERT INTO domain_metrics (DomainName, Count) SELECT RequestDomain, 1 FROM buffer WHERE true ON CONFLICT (DomainName) DO UPDATE SET Count = Count + 1",
		"INSERT INTO query_metrics (QueryType, Count) SELECT RequestType, 1 FROM buffer WHERE true ON CONFLICT (QueryType) DO UPDATE SET Count = Count + 1",
		"INSERT INTO client_names (Address, ClientName) SELECT DISTINCT Address, ClientName FROM buffer WHERE ClientName != '' ON CONFLICT(Address) DO UPDATE SET ClientName = (SELECT ClientName FROM (SELECT b.ClientName, length(b.ClientName) as Length FROM buffer b WHERE b.ClientName != '' AND Address = b.Address UNION SELECT c.ClientName, length(c.ClientName) as Length FROM client_names c where Address = c.Address ORDER BY Length DESC LIMIT 1))",
	}

	// start executing statements
	for _, s := range stmts {
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
	defer rows.Close()
	if rows == nil {
		return []*TopInfo{}
	}

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
	return metrics.top("SELECT CASE WHEN n.ClientName = '' THEN c.Address ELSE n.ClientName END AS Name, Count FROM client_metrics c JOIN client_names n ON c.Address = n.Address ORDER BY Count DESC", limit)
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
