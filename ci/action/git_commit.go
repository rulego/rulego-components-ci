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
	"errors"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/maps"
	"github.com/rulego/rulego/utils/str"
	"time"
)

func init() {
	_ = rulego.Registry.Register(&GitCommitNode{})
}

// GitCommitNodeConfiguration 节点配置
type GitCommitNodeConfiguration struct {
	// 本地目录
	Directory string
	// 添加的文件模式匹配
	Pattern string
	// 注释消息
	Message string
	//签名
	Signature Signature
}

// GitCommitNode 实现 Git 推送
type GitCommitNode struct {
	baseGitNode
	// 节点配置
	Config GitCommitNodeConfiguration
	hasVar bool
}

// Type 组件类型
func (x *GitCommitNode) Type() string {
	return "ci/gitCommit"
}

func (x *GitCommitNode) New() types.Node {
	return &GitCommitNode{Config: GitCommitNodeConfiguration{}}
}

// Init 初始化
func (x *GitCommitNode) Init(ruleConfig types.Config, configuration types.Configuration) error {
	err := maps.Map2Struct(configuration, &x.Config)
	err = maps.Map2Struct(configuration, &x.baseGitNode.Config)
	if str.CheckHasVar(x.Config.Directory) || str.CheckHasVar(x.Config.Pattern) || str.CheckHasVar(x.Config.Signature.AuthorName) || str.CheckHasVar(x.Config.Signature.AuthorEmail) {
		x.hasVar = true
	}
	return err
}

// OnMsg 处理消息
func (x *GitCommitNode) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	var evn map[string]interface{}
	if x.hasVar {
		evn = base.NodeUtils.GetEvnAndMetadata(ctx, msg)
	}
	workDir := x.getWorkDir(msg, evn)
	msg.Metadata.PutValue(KeyWorkDir, workDir)
	// 打开仓库
	r, err := git.PlainOpen(workDir)
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	// 创建并提交更改
	w, err := r.Worktree()
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	// 检查是否有文件更改
	status, err := w.Status()
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	if status.IsClean() {
		ctx.TellFailure(msg, errors.New("no changes to commit"))
	} else {
		//添加文件
		err = w.AddGlob(x.getPattern(msg, evn))
		if err != nil {
			ctx.TellFailure(msg, err)
			return
		}
		commit, err := w.Commit(x.getMessage(msg, evn), &git.CommitOptions{
			Author: &object.Signature{
				Name:  x.getSignatureName(msg, evn),
				Email: x.getSignatureEmail(msg, evn),
				When:  time.Now(),
			},
		})
		if err != nil {
			ctx.TellFailure(msg, err)
			return
		}
		msg.Metadata.PutValue(KeyHash, commit.String())
		ctx.TellSuccess(msg)
	}
}

// Destroy 销毁
func (x *GitCommitNode) Destroy() {
}

func (x *GitCommitNode) getPattern(_ types.RuleMsg, evn map[string]interface{}) string {
	pattern := x.Config.Pattern
	if evn != nil {
		pattern = str.ExecuteTemplate(pattern, evn)
	}
	return pattern
}

func (x *GitCommitNode) getMessage(_ types.RuleMsg, evn map[string]interface{}) string {
	message := x.Config.Message
	if evn != nil {
		message = str.ExecuteTemplate(message, evn)
	}
	return message
}

func (x *GitCommitNode) getSignatureName(_ types.RuleMsg, evn map[string]interface{}) string {
	name := x.Config.Signature.AuthorName
	if evn != nil {
		name = str.ExecuteTemplate(name, evn)
	}
	return name
}

func (x *GitCommitNode) getSignatureEmail(_ types.RuleMsg, evn map[string]interface{}) string {
	email := x.Config.Signature.AuthorEmail
	if evn != nil {
		email = str.ExecuteTemplate(email, evn)
	}
	return email
}
