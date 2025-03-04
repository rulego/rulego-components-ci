/*
 * Copyright 2024 The RuleGo Authors.
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
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/maps"
	"github.com/rulego/rulego/utils/str"
)

func init() {
	_ = rulego.Registry.Register(&GitLogNode{})
}

// GitLogNodeConfiguration 节点配置
type GitLogNodeConfiguration struct {
	// 本地目录
	Directory string `json:"directory"`
	// 日志数量限制
	Limit int `json:"limit"`
	// 起始时间，格式：yyyy-MM-dd 或者 yyyy-MM-dd HH:mm:ss 如 "2006-01-02 15:04:05"
	StartTime string `json:"startTime"`
	// 结束时间，格式：yyyy-MM-dd 或者 yyyy-MM-dd HH:mm:ss 如 "2006-01-02 15:04:05"
	EndTime string `json:"endTime"`
}

// GitLogNode 实现获取 Git 日志
type GitLogNode struct {
	baseGitNode
	// 节点配置
	Config            GitLogNodeConfiguration
	hasVar            bool
	startTimeTemplate str.Template
	endTimeTemplate   str.Template
}

// Type 组件类型
func (x *GitLogNode) Type() string {
	return "ci/gitLog"
}

func (x *GitLogNode) New() types.Node {
	return &GitLogNode{Config: GitLogNodeConfiguration{
		Limit: 10,
	}}
}

// Init 初始化
func (x *GitLogNode) Init(ruleConfig types.Config, configuration types.Configuration) error {
	err := maps.Map2Struct(configuration, &x.Config)
	err = maps.Map2Struct(configuration, &x.baseGitNode.Config)
	x.Config.StartTime = strings.TrimSpace(x.Config.StartTime)
	x.Config.EndTime = strings.TrimSpace(x.Config.EndTime)

	// 检查时间格式并格式化
	x.startTimeTemplate = str.NewTemplate(x.Config.StartTime)
	x.endTimeTemplate = str.NewTemplate(x.Config.EndTime)
	if str.CheckHasVar(x.Config.StartTime) || str.CheckHasVar(x.Config.EndTime) {
		x.hasVar = true
	}
	return err
}

// OnMsg 处理消息
func (x *GitLogNode) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	var evn map[string]interface{}
	if x.hasVar {
		evn = base.NodeUtils.GetEvnAndMetadata(ctx, msg)
	}
	workDir := x.getWorkDir(msg, evn)
	msg.Metadata.PutValue(KeyWorkDir, workDir)
	// 动态解析配置
	startTimeStr := x.startTimeTemplate.Execute(evn)
	endTimeStr := x.endTimeTemplate.Execute(evn)

	if startTimeStr != "" && len(startTimeStr) == 10 {
		startTimeStr = startTimeStr + " 00:00:00"
	}
	if endTimeStr != "" && len(endTimeStr) == 10 {
		endTimeStr = endTimeStr + " 23:59:59"
	}
	// 打开仓库
	r, err := git.PlainOpen(workDir)
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}

	// 解析起始时间和结束时间
	startTime, _ := time.Parse("2006-01-02 15:04:05", startTimeStr)
	endTime, _ := time.Parse("2006-01-02 15:04:05", endTimeStr)

	// 获取日志迭代器
	iter, err := r.Log(&git.LogOptions{
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	defer iter.Close()

	var logs []*object.Commit
	var messages []LogMsg
	limit := x.Config.Limit
	//if limit == 0 {
	//	limit = 10 // 默认限制为 10 条日志
	//}
	i := 0
	for {
		commit, err := iter.Next()
		if err != nil {
			break
		}
		// 检查时间范围
		if (!startTime.IsZero() && commit.Committer.When.Before(startTime)) ||
			(!endTime.IsZero() && commit.Committer.When.After(endTime)) {
			continue
		}
		logs = append(logs, commit)
		messages = append(messages, LogMsg{
			Hash: commit.Hash.String(),
			Author: Committer{
				Name:  commit.Author.Name,
				Email: commit.Author.Email,
				When:  commit.Author.When,
			},
			Committer: Committer{
				Name:  commit.Committer.Name,
				Email: commit.Committer.Email,
				When:  commit.Committer.When,
			},
			MergeTag: commit.MergeTag,
			Message:  commit.Message,
			TreeHash: commit.TreeHash.String(),
			Encoding: commit.Encoding,
		})
		i++
		if limit != 0 && i >= limit {
			break
		}
	}

	// 将日志添加到消息中
	msg.DataType = types.JSON
	msg.Data = str.ToString(messages)
	ctx.TellSuccess(msg)
}

// Destroy 销毁
func (x *GitLogNode) Destroy() {
}

type LogMsg struct {
	// Hash of the commit object.
	Hash string `json:"hash"`
	// Author is the original author of the commit.
	Author Committer `json:"author"`
	// Committer is the one performing the commit, might be different from
	// Author.
	Committer Committer `json:"committer"`
	// MergeTag is the embedded tag object when a merge commit is created by
	// merging a signed tag.
	MergeTag string `json:"mergeTag"`
	// Message is the commit message, contains arbitrary text.
	Message string `json:"message"`
	// TreeHash is the hash of the root tree of the commit.
	TreeHash string `json:"treeHash"`
	// Encoding is the encoding of the commit.
	Encoding object.MessageEncoding `json:"encoding"`
}
type Committer struct {
	// Name represents a person name. It is an arbitrary string.
	Name string `json:"name"`
	// Email is an email, but it cannot be assumed to be well-formed.
	Email string `json:"email"`
	// When is the timestamp of the signature.
	When time.Time `json:"when"`
}
