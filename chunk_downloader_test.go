package gosnowflake

import (
	"context"
	"database/sql/driver"
	"testing"
)

func TestChunkDownloaderDoesNotStartWhenArrowParsingCausesError(t *testing.T) {
	tcs := []string{
		"invalid base64",
		"aW52YWxpZCBhcnJvdw==", // valid base64, but invalid arrow
	}
	for _, tc := range tcs {
		t.Run(tc, func(t *testing.T) {
			scd := snowflakeChunkDownloader{
				ctx:               context.Background(),
				QueryResultFormat: "arrow",
				RowSet: rowSetType{
					RowSetBase64: tc,
				},
			}

			err := scd.start()

			assertNotNilF(t, err)
		})
	}
}

func TestWithArrowBatchesWhenQueryReturnsNoRowsWhenUsingNativeGoSQLInterface(t *testing.T) {
	runDBTest(t, func(dbt *DBTest) {
		var rows driver.Rows
		var err error
		err = dbt.conn.Raw(func(x interface{}) error {
			rows, err = x.(driver.QueryerContext).QueryContext(WithArrowBatches(context.Background()), "SELECT 1 WHERE 0 = 1", nil)
			return err
		})
		assertNilF(t, err)
		rows.Close()
	})
}

func TestWithArrowBatchesWhenQueryReturnsRowsAndReadingRows(t *testing.T) {
	runDBTest(t, func(dbt *DBTest) {
		rows := dbt.mustQueryContext(WithArrowBatches(context.Background()), "SELECT 1")
		defer rows.Close()
		assertFalseF(t, rows.Next())
	})
}

func TestWithArrowBatchesWhenQueryReturnsNoRowsAndReadingRows(t *testing.T) {
	runDBTest(t, func(dbt *DBTest) {
		rows := dbt.mustQueryContext(WithArrowBatches(context.Background()), "SELECT 1 WHERE 1 = 0")
		defer rows.Close()
		assertFalseF(t, rows.Next())
	})
}

func TestWithArrowBatchesWhenQueryReturnsNoRowsAndReadingArrowBatches(t *testing.T) {
	runDBTest(t, func(dbt *DBTest) {
		var rows driver.Rows
		var err error
		err = dbt.conn.Raw(func(x any) error {
			rows, err = x.(driver.QueryerContext).QueryContext(WithArrowBatches(context.Background()), "SELECT 1 WHERE 1 = 0", nil)
			return err
		})
		assertNilF(t, err)
		defer rows.Close()
		batches, err := rows.(SnowflakeRows).GetArrowBatches()
		assertNilF(t, err)
		assertEmptyE(t, batches)
	})
}

func TestWithArrowBatchesWhenQueryReturnsSomeRowsInGivenFormatUsingNativeGoSQLInterface(t *testing.T) {
	for _, tc := range []struct {
		useJSON bool
		desc    string
	}{
		{
			useJSON: true,
			desc:    "json",
		},
		{
			useJSON: false,
			desc:    "arrow",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			runDBTest(t, func(dbt *DBTest) {
				if tc.useJSON {
					dbt.mustExec(forceJSON)
				}
				var rows driver.Rows
				var err error
				err = dbt.conn.Raw(func(x interface{}) error {
					rows, err = x.(driver.QueryerContext).QueryContext(WithArrowBatches(context.Background()), "SELECT 1", nil)
					return err
				})
				assertNilF(t, err)
				defer func() {
					assertNilF(t, rows.Close())
				}()
				values := make([]driver.Value, 1)
				assertNotNilE(t, rows.Next(values)) // we deliberately check that there is an error, because we are in arrow batches mode
				assertEqualE(t, values[0], nil)
			})
		})
	}
}

func TestWithArrowBatchesWhenQueryReturnsSomeRowsInGivenFormat(t *testing.T) {
	for _, tc := range []struct {
		useJSON bool
		desc    string
	}{
		{
			useJSON: true,
			desc:    "json",
		},
		{
			useJSON: false,
			desc:    "arrow",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			runDBTest(t, func(dbt *DBTest) {
				if tc.useJSON {
					dbt.mustExec(forceJSON)
				}
				rows := dbt.mustQueryContext(WithArrowBatches(context.Background()), "SELECT 1")
				defer func() {
					assertNilF(t, rows.Close())
				}()
				assertFalseF(t, rows.Next())
			})
		})
	}
}
