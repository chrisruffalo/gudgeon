package metrics

import (
	"database/sql"
	"fmt"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/rule"
)

// info from db
type TopInfo struct {
	Desc  string
	Count uint64
}

func (metrics *metrics) updateDetailedMetrics(info *metricsInfo) {
	// if no db, can't record, or if the detailed metrics feature is disabled don't record
	if metrics.db == nil  || !*metrics.config.Metrics.Detailed {
		return
	}

	// create transaction
	tx, err := metrics.db.Begin()
	if err != nil {
		log.Errorf("Could not begin transaction: %s", err)
		return
	}
	defer tx.Rollback()

	// address/client information
	if info.address != "" {
		metrics.incrementClientHit(tx, info.address)
	}

	// request metrics
	if info.request != nil {
		// domain hit
		domain := info.request.Question[0].Name
		metrics.incrementDomainHit(tx, domain)

		// request type hit
		qtype := dns.Type(info.request.Question[0].Qtype).String()
		metrics.incrementTypeHit(tx, qtype)
	}

	// list/rule information
	if info.result != nil && (info.result.Match == rule.MatchBlock || info.result.Match == rule.MatchAllow) && info.result.MatchList != nil {
		metrics.incrementListHit(tx, info.result.MatchList)
		if "" != info.result.MatchRule {
			metrics.incrementRuleHit(tx, info.result.MatchList, info.result.MatchRule)
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Errorf("Commiting detailed metrics: %s", err)
	}
}

// todo: share some code here
func (metrics *metrics) mkStmt(tx *sql.Tx, stmt string) *sql.Stmt {
	pstmt, found := metrics.stmtCache[stmt]

	if !found {
		var err error
		pstmt, err = tx.Prepare(stmt)
		if err != nil {
			log.Errorf("Preparing statement: %s",  err)
			return nil
		}
		metrics.stmtCache[stmt] = pstmt
	}

	return pstmt
}


func (metrics *metrics) incrementListHit(tx *sql.Tx, list *config.GudgeonList) {
	pstmt := metrics.mkStmt(tx, "INSERT INTO lists (Name, ShortName, Hits) VALUES (?, ?, 1) ON CONFLICT(ShortName) DO UPDATE SET Hits = Hits + 1")
	if pstmt == nil {
		return
	}

	_, err := tx.Stmt(pstmt).Exec(list.CanonicalName(), list.ShortName())
	if err != nil {
		log.Errorf("Incrementing list hit metrics: %s", err)
	}
}

func (metrics *metrics) incrementRuleHit(tx *sql.Tx, list *config.GudgeonList, ruleText string) {
	pstmt := metrics.mkStmt(tx, "INSERT INTO rule_hits (ListId, Rule, Hits) VALUES ((SELECT Id FROM lists WHERE ShortName = ? LIMIT 1), ?,10) ON CONFLICT(ListId, Rule) DO UPDATE SET Hits = Hits + 1")
	if pstmt == nil {
		return
	}

	_, err := tx.Stmt(pstmt).Exec(list.ShortName(), ruleText)
	if err != nil {
		log.Errorf("Incrementing rule hit metrics: %s", err)
	}
}

func (metrics *metrics) incrementClientHit(tx *sql.Tx, clientAddress string) {
	pstmt := metrics.mkStmt(tx, "INSERT INTO client_metrics (Address, Count) VALUES (?, 1) ON CONFLICT (Address) DO UPDATE SET Count = Count + 1")
	if pstmt == nil {
		return
	}

	_, err := tx.Stmt(pstmt).Exec(clientAddress)
	if err != nil {
		log.Errorf("Incrementing client hit metrics: %s", err)
	}
}

func (metrics *metrics) incrementDomainHit(tx *sql.Tx, domain string) {
	pstmt := metrics.mkStmt(tx, "INSERT INTO domain_metrics (DomainName, Count) VALUES (?, 1) ON CONFLICT (DomainName) DO UPDATE SET Count = Count + 1")
	if pstmt == nil {
		return
	}

	_, err := tx.Stmt(pstmt).Exec(domain)
	if err != nil {
		log.Errorf("Incrementing domain hit metrics: %s", err)
	}
}

func (metrics *metrics) incrementTypeHit(tx *sql.Tx, queryType string) {
	pstmt := metrics.mkStmt(tx, "INSERT INTO query_types (QueryType, Count) VALUES (?, 1) ON CONFLICT (QueryType) DO UPDATE SET Count = Count + 1")
	if pstmt == nil {
		return
	}

	_, err := tx.Stmt(pstmt).Exec(queryType)
	if err != nil {
		log.Errorf("Incrementing type hit metrics: %s", err)
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