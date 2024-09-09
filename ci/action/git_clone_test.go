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
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/test"
	"github.com/rulego/rulego/test/assert"
	"testing"
)

func TestGitCloneNode(t *testing.T) {
	Registry := &types.SafeComponentSlice{}
	Registry.Add(&GitCloneNode{})
	var targetNodeType = "ci/gitClone"

	t.Run("NewNode", func(t *testing.T) {
		test.NodeNew(t, targetNodeType, &GitCloneNode{}, types.Configuration{}, Registry)
	})

	t.Run("InitNode", func(t *testing.T) {
		node, _ := test.CreateAndInitNode(targetNodeType, types.Configuration{
			"repository": "",
			"directory":  "",
			"reference":  "",
			"authType":   "ssh",
		}, Registry)
		metaData := types.BuildMetadata(make(map[string]string))
		metaData.PutValue(KeyWorkDir, "d://")
		metaData.PutValue(KeyRef, "main")
		metaData.PutValue(KeyGitSshUrl, "git@github.com:rulego/rulego-components-ci.git")
		metaData.PutValue(KeyGitHttpUrl, "https://github.com/rulego/rulego-components-ci")
		msg := types.NewMsg(0, "test", types.JSON, metaData, "")
		evn := base.NodeUtils.GetEvnAndMetadata(nil, msg)
		workDir := (node.(*GitCloneNode)).getWorkDir(msg, evn)
		repository := (node.(*GitCloneNode)).getRepository(msg, evn)
		reference := (node.(*GitCloneNode)).getReferenceName(msg, evn)
		assert.Equal(t, "d:/rulego-components-ci", workDir)
		assert.Equal(t, "git@github.com:rulego/rulego-components-ci.git", repository)
		assert.Equal(t, "main", reference)

		node, _ = test.CreateAndInitNode(targetNodeType, types.Configuration{
			"repository": "",
			"directory":  "",
			"reference":  "",
			"authType":   "token",
		}, Registry)
		repository = (node.(*GitCloneNode)).getRepository(msg, evn)
		assert.Equal(t, "https://github.com/rulego/rulego-components-ci", repository)

		node, _ = test.CreateAndInitNode(targetNodeType, types.Configuration{
			"repository": "${metadata.gitHttpUrl}",
			"directory":  "${metadata.workDir}",
			"reference":  "${metadata.ref}",
			"authType":   "token",
		}, Registry)
		repository = (node.(*GitCloneNode)).getRepository(msg, evn)
		assert.Equal(t, "d:/rulego-components-ci", workDir)
		assert.Equal(t, "https://github.com/rulego/rulego-components-ci", repository)
		assert.Equal(t, "main", reference)
	})

}
