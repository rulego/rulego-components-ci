/*
 * Copyright 2023 The RuleGo Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package action

import (
	"encoding/json"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/test"
	"github.com/rulego/rulego/test/assert"
	"testing"
	"time"
)

func TestPsNode(t *testing.T) {
	Registry := &types.SafeComponentSlice{}
	Registry.Add(&PsNode{})
	var targetNodeType = "ci/ps"

	t.Run("NewNode", func(t *testing.T) {
		test.NodeNew(t, targetNodeType, &PsNode{}, types.Configuration{}, Registry)
	})

	t.Run("InitNode", func(t *testing.T) {
		test.NodeInit(t, targetNodeType, types.Configuration{
			"options": []string{OptionsHostInfo},
		}, types.Configuration{
			"options": []string{OptionsHostInfo},
		}, Registry)
	})

	t.Run("OnMsg", func(t *testing.T) {
		node1, err := test.CreateAndInitNode(targetNodeType, types.Configuration{
			"options": []string{OptionsHostInfo},
		}, Registry)
		assert.Nil(t, err)
		node2, err := test.CreateAndInitNode(targetNodeType, types.Configuration{
			"options": nil,
		}, Registry)
		assert.Nil(t, err)

		metaData := types.BuildMetadata(make(map[string]string))
		msgList := []test.Msg{
			{
				MetaData: metaData,
				MsgType:  "queryServerMetrics",
			},
		}

		var nodeList = []test.NodeAndCallback{
			{
				Node:    node1,
				MsgList: msgList,
				Callback: func(msg types.RuleMsg, relationType string, err error) {
					result := make(map[string]interface{})
					_ = json.Unmarshal([]byte(msg.GetData()), &result)
					_, ok := result[OptionsHostInfo]
					assert.True(t, ok)
					assert.Equal(t, types.Success, relationType)
				},
			},
			{
				Node:    node2,
				MsgList: msgList,
				Callback: func(msg types.RuleMsg, relationType string, err error) {
					result := make(map[string]interface{})
					_ = json.Unmarshal([]byte(msg.GetData()), &result)
					_, ok := result[OptionsHostInfo]
					assert.True(t, ok)
					_, ok = result[OptionsCpuInfo]
					assert.True(t, ok)
					_, ok = result[OptionsCpuPercent]
					assert.True(t, ok)
					_, ok = result[OptionsVirtualMemory]
					assert.True(t, ok)
					_, ok = result[OptionsVirtualMemory]
					assert.True(t, ok)
					_, ok = result[OptionsSwapMemory]
					assert.True(t, ok)
					_, ok = result[OptionsDiskUsage]
					assert.True(t, ok)
					_, ok = result[OptionsDiskIOCounters]
					assert.True(t, ok)
					_, ok = result[OptionsNetIOCounters]
					assert.True(t, ok)
					_, ok = result[OptionsInterfaces]
					assert.True(t, ok)
				},
			},
		}
		for _, item := range nodeList {
			test.NodeOnMsgWithChildren(t, item.Node, item.MsgList, item.ChildrenNodes, item.Callback)
		}
		time.Sleep(time.Second * 5)
	})
}
