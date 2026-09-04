package db

import (
	"context"
	"database/sql/driver"
	"log/slog"
	"strings"
	"time"

	"github.com/lib/pq"
)

// The slow-query log wraps lib/pq at the database/sql/driver layer, so every
// repository query is timed without touching the 70 repository files or
// changing how they use *sql.DB. Only statements over the threshold are
// logged, with the statement text (whitespace-collapsed, truncated) and never
// the arguments — customer data must not land in logs.

// slowQueryTextLimit caps the logged statement length.
const slowQueryTextLimit = 200

// newSlowLogConnector wraps a pq connector so every connection it opens is
// timed. threshold <= 0 disables logging (the connector is returned unwrapped).
func newSlowLogConnector(dsn string, threshold time.Duration, logger *slog.Logger) (driver.Connector, error) {
	base, err := pq.NewConnector(dsn)
	if err != nil {
		return nil, err
	}
	if threshold <= 0 {
		return base, nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &slowLogConnector{Connector: base, threshold: threshold, logger: logger}, nil
}

type slowLogConnector struct {
	driver.Connector
	threshold time.Duration
	logger    *slog.Logger
}

func (c *slowLogConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &slowLogConn{Conn: conn, c: c}, nil
}

// observe logs one statement if it ran longer than the threshold.
func (c *slowLogConnector) observe(ctx context.Context, kind, query string, start time.Time, err error) {
	elapsed := time.Since(start)
	if elapsed < c.threshold {
		return
	}
	attrs := []any{
		"kind", kind,
		"duration_ms", elapsed.Milliseconds(),
		"threshold_ms", c.threshold.Milliseconds(),
		"query", compactQuery(query),
	}
	if err != nil {
		attrs = append(attrs, "error", err)
	}
	c.logger.WarnContext(ctx, "slow query", attrs...)
}

// compactQuery collapses whitespace and truncates, so multi-line SQL logs as
// one readable line.
func compactQuery(q string) string {
	q = strings.Join(strings.Fields(q), " ")
	if len(q) > slowQueryTextLimit {
		return q[:slowQueryTextLimit] + "…"
	}
	return q
}

// slowLogConn forwards every optional driver interface pq implements so
// database/sql keeps taking the same fast paths (context-aware exec/query,
// named-value checking, session reset, validation) it would with bare pq.
type slowLogConn struct {
	driver.Conn
	c *slowLogConnector
}

var (
	_ driver.QueryerContext     = (*slowLogConn)(nil)
	_ driver.ExecerContext      = (*slowLogConn)(nil)
	_ driver.ConnPrepareContext = (*slowLogConn)(nil)
	_ driver.ConnBeginTx        = (*slowLogConn)(nil)
	_ driver.Pinger             = (*slowLogConn)(nil)
	_ driver.SessionResetter    = (*slowLogConn)(nil)
	_ driver.Validator          = (*slowLogConn)(nil)
	_ driver.NamedValueChecker  = (*slowLogConn)(nil)
)

func (s *slowLogConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	start := time.Now()
	rows, err := s.Conn.(driver.QueryerContext).QueryContext(ctx, query, args)
	s.c.observe(ctx, "query", query, start, err)
	return rows, err
}

func (s *slowLogConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	start := time.Now()
	res, err := s.Conn.(driver.ExecerContext).ExecContext(ctx, query, args)
	s.c.observe(ctx, "exec", query, start, err)
	return res, err
}

func (s *slowLogConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	stmt, err := s.Conn.(driver.ConnPrepareContext).PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return &slowLogStmt{Stmt: stmt, query: query, c: s.c}, nil
}

func (s *slowLogConn) Prepare(query string) (driver.Stmt, error) {
	return s.PrepareContext(context.Background(), query)
}

func (s *slowLogConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return s.Conn.(driver.ConnBeginTx).BeginTx(ctx, opts)
}

func (s *slowLogConn) Ping(ctx context.Context) error {
	return s.Conn.(driver.Pinger).Ping(ctx)
}

func (s *slowLogConn) ResetSession(ctx context.Context) error {
	return s.Conn.(driver.SessionResetter).ResetSession(ctx)
}

func (s *slowLogConn) IsValid() bool {
	return s.Conn.(driver.Validator).IsValid()
}

func (s *slowLogConn) CheckNamedValue(nv *driver.NamedValue) error {
	return s.Conn.(driver.NamedValueChecker).CheckNamedValue(nv)
}

// slowLogStmt times prepared-statement executions, which is also the path
// pq.CopyIn takes.
type slowLogStmt struct {
	driver.Stmt
	query string
	c     *slowLogConnector
}

var (
	_ driver.StmtExecContext  = (*slowLogStmt)(nil)
	_ driver.StmtQueryContext = (*slowLogStmt)(nil)
)

// pq's regular statements implement the *Context variants, but its CopyIn
// statement only implements the legacy Exec/Query, so fall back the way
// database/sql itself does (positional values only; pq has no named args).
func (s *slowLogStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	start := time.Now()
	var (
		res driver.Result
		err error
	)
	if sc, ok := s.Stmt.(driver.StmtExecContext); ok {
		res, err = sc.ExecContext(ctx, args)
	} else {
		res, err = s.Stmt.Exec(namedToValues(args)) //nolint:staticcheck // legacy path is the only one pq's CopyIn statement offers
	}
	s.c.observe(ctx, "stmt_exec", s.query, start, err)
	return res, err
}

func (s *slowLogStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	start := time.Now()
	var (
		rows driver.Rows
		err  error
	)
	if sc, ok := s.Stmt.(driver.StmtQueryContext); ok {
		rows, err = sc.QueryContext(ctx, args)
	} else {
		rows, err = s.Stmt.Query(namedToValues(args)) //nolint:staticcheck // legacy path is the only one pq's CopyIn statement offers
	}
	s.c.observe(ctx, "stmt_query", s.query, start, err)
	return rows, err
}

func namedToValues(args []driver.NamedValue) []driver.Value {
	vals := make([]driver.Value, len(args))
	for i, a := range args {
		vals[i] = a.Value
	}
	return vals
}
