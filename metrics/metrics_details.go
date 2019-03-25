package metrics

import (
	"fmt"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/rule"
)

// info from db
type TopInfo struct {
	Desc  string
	Count uint64
}

func (metrics *metrics) updateDetailedMetrics(info *metricsInfo) {
	// if no db, can't record, or if the detailed metrics feature is disabled don't record
	if metrics.db == nil || !*metrics.config.Metrics.Detailed {
		return
	}

	if metrics.detailTx == nil {
		var err error
		metrics.detailTx, err = metrics.db.Begin()
		if err != nil {
			metrics.detailTx = nil
			log.Errorf("Starting metrics insert transaction: %s", err)
			return
		}
	}

	// prepare statement
	if metrics.detailStmt == nil {
		var err error
		metrics.detailStmt, err = metrics.detailTx.Prepare("INSERT INTO metrics_temp (Address, QueryDomain, QueryType, ListName, ListShortName, Rule) VALUES (?, ?, ?, ?, ?, ?)")
		if err != nil {
			log.Errorf("Preparing statement: %s", err)
			return
		}
	}

	// query params
	queryName := ""
	queryType := ""
	listName := ""
	listShortName := ""
	ruleText := ""

	// conditioanlly add domain details
	if info.request != nil {
		queryName = info.request.Question[0].Name
		queryType = dns.Type(info.request.Question[0].Qtype).String()
	}

	// conditionally add details
	if info.result != nil && (info.result.Match == rule.MatchBlock || info.result.Match == rule.MatchAllow) && info.result.MatchList != nil {
		listName = info.result.MatchList.CanonicalName()
		listShortName = info.result.MatchList.ShortName()
		ruleText = info.result.MatchRule
	}

	_, err := metrics.detailTx.Stmt(metrics.detailStmt).Exec(info.address, queryName, queryType, listName, listShortName, ruleText)
	if err != nil {
		log.Errorf("Inserting metric temp record: %s", err)
		return
	}
}

func (metrics *metrics) flushDetailedMetrics() {
	if metrics.detailTx != nil {
		defer metrics.detailTx.Rollback()
		err := metrics.detailTx.Commit()
		metrics.detailTx = nil
		if err != nil {
			log.Errorf("Could not commit pending metrics temp entries: %s", err)
			return
		}
	}

	// start a new for moving fields
	tx, err := metrics.db.Begin()
	if err != nil {
		log.Errorf("Creating detailed metrics transaction: %s", err)
		return
	}
	defer tx.Rollback()

	// statements
	stmts := []string{
		"INSERT INTO lists (Name, ShortName, Hits) SELECT ListName, ListShortName, 1 FROM metrics_temp WHERE ListShortName != '' ON CONFLICT(ShortName) DO UPDATE SET Hits = Hits + 1",
		"INSERT INTO rule_hits (ListId, Rule, Hits) SELECT l.Id, m.Rule, 1 FROM metrics_temp m JOIN lists l ON m.ListShortName = l.ShortName WHERE m.Rule != '' ON CONFLICT (ListId, Rule) DO UPDATE SET Hits = Hits + 1",
		"INSERT INTO client_metrics (Address, Count) SELECT Address, 1 FROM metrics_temp WHERE true ON CONFLICT (Address) DO UPDATE SET Count = Count + 1",
		"INSERT INTO domain_metrics (DomainName, Count) SELECT QueryDomain, 1 FROM metrics_temp WHERE true ON CONFLICT (DomainName) DO UPDATE SET Count = Count + 1",
		"INSERT INTO query_types (QueryType, Count) SELECT QueryType, 1 FROM metrics_temp WHERE true ON CONFLICT (QueryType) DO UPDATE SET Count = Count + 1",
		"DELETE FROM metrics_temp", // when done, drop all temp records
	}

	// start executing statements
	for _, s := range stmts {
		_, err := tx.Exec(s)
		if err != nil {
			log.Errorf("Moving metrics (stmt=\"%s\") data: %s", s, err)
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Errorf("Committing metrics flush: %s", err)
	}
}

func (metrics *metrics) top(stmt string, limit int) []*TopInfo {
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
	return metrics.top("SELECT Address, Count FROM client_metrics ORDER BY Count DESC", limit)
}

func (metrics *metrics) TopDomains(limit int) []*TopInfo {
	return metrics.top("SELECT DomainName, Count FROM domain_metrics ORDER BY Count DESC", limit)
}

func (metrics *metrics) TopQueryTypes(limit int) []*TopInfo {
	return metrics.top("SELECT QueryType, Count FROM query_types ORDER BY Count DESC", limit)
}

func (metrics *metrics) TopLists(limit int) []*TopInfo {
	return metrics.top("SELECT Name, Hits FROM lists ORDER BY Hits DESC", limit)
}

func (metrics *metrics) TopRules(limit int) []*TopInfo {
	return metrics.top("SELECT r.Rule, r.Hits FROM rule_hits r JOIN lists l ON l.Id = r.ListId ORDER BY r.Hits DESC", limit)
}
