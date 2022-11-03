// SPDX-License-Identifier: AGPL-3.0-only

package compactor

import (
	"context"
	"fmt"
	"testing"

	"github.com/oklog/ulid"
	"github.com/prometheus/prometheus/tsdb"
	"github.com/stretchr/testify/assert"

	"github.com/grafana/mimir/pkg/storage/tsdb/metadata"
)

func TestSplitAndMergePlanner_Plan(t *testing.T) {
	block1 := ulid.MustNew(1, nil)
	block2 := ulid.MustNew(2, nil)
	block3 := ulid.MustNew(3, nil)

	tests := map[string]struct {
		ranges          []int64
		blocksByMinTime []*metadata.Meta
		expectedErr     error
	}{
		"no blocks": {
			ranges:          []int64{20, 40, 60},
			blocksByMinTime: []*metadata.Meta{},
		},
		"a source block is larger then the largest range": {
			ranges: []int64{20, 40, 60},
			blocksByMinTime: []*metadata.Meta{
				{BlockMeta: tsdb.BlockMeta{ULID: block1, MinTime: 0, MaxTime: 20, Version: metadata.TSDBVersion1}},
				{BlockMeta: tsdb.BlockMeta{ULID: block2, MinTime: 10, MaxTime: 80, Version: metadata.TSDBVersion1}},
				{BlockMeta: tsdb.BlockMeta{ULID: block3, MinTime: 12, MaxTime: 15, Version: metadata.TSDBVersion1}},
			},
			expectedErr: fmt.Errorf("block %s with time range 10:80 is outside the largest expected range 0:60",
				block2.String()),
		},
		"source blocks are smaller then the largest range but compacted block is larger": {
			ranges: []int64{20, 40, 60},
			blocksByMinTime: []*metadata.Meta{
				{BlockMeta: tsdb.BlockMeta{ULID: block1, MinTime: 10, MaxTime: 20, Version: metadata.TSDBVersion1}},
				{BlockMeta: tsdb.BlockMeta{ULID: block2, MinTime: 30, MaxTime: 40, Version: metadata.TSDBVersion1}},
				{BlockMeta: tsdb.BlockMeta{ULID: block3, MinTime: 50, MaxTime: 70, Version: metadata.TSDBVersion1}},
			},
			expectedErr: fmt.Errorf("block %s with time range 50:70 is outside the largest expected range 0:60",
				block3.String()),
		},
		"source blocks and compacted block are smaller then the largest range but misaligned": {
			ranges: []int64{20, 40, 60},
			blocksByMinTime: []*metadata.Meta{
				{BlockMeta: tsdb.BlockMeta{ULID: block1, MinTime: 50, MaxTime: 70, Version: metadata.TSDBVersion1}},
				{BlockMeta: tsdb.BlockMeta{ULID: block2, MinTime: 70, MaxTime: 80, Version: metadata.TSDBVersion1}},
			},
			expectedErr: fmt.Errorf("block %s with time range 50:70 is outside the largest expected range 0:60",
				block1.String()),
		},
		"blocks fit within the largest range": {
			ranges: []int64{20, 40, 60},
			blocksByMinTime: []*metadata.Meta{
				{BlockMeta: tsdb.BlockMeta{ULID: block1, MinTime: 10, MaxTime: 20, Version: metadata.TSDBVersion1}},
				{BlockMeta: tsdb.BlockMeta{ULID: block2, MinTime: 20, MaxTime: 40, Version: metadata.TSDBVersion1}},
				{BlockMeta: tsdb.BlockMeta{ULID: block3, MinTime: 20, MaxTime: 60, Version: metadata.TSDBVersion1}},
			},
		},
	}

	for testName, testData := range tests {
		t.Run(testName, func(t *testing.T) {
			c := NewSplitAndMergePlanner(testData.ranges)
			actual, err := c.Plan(context.Background(), testData.blocksByMinTime)
			assert.Equal(t, testData.expectedErr, err)

			if testData.expectedErr == nil {
				// Since the planner is a pass-through we do expect to get the same input blocks on success.
				assert.Equal(t, testData.blocksByMinTime, actual)
			}
		})
	}
}
