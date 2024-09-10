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
	_ = rulego.Registry.Register(&GitCreateTagNode{})
}

// GitCreateTagNodeConfiguration 节点配置
type GitCreateTagNodeConfiguration struct {
	// 本地目录
	Directory string
	// 标签名称
	Tag string
	// 注释消息
	Message string
	//签名
	Signature Signature
}

// GitCreateTagNode 实现 Git 推送
type GitCreateTagNode struct {
	baseGitNode
	// 节点配置
	Config GitCreateTagNodeConfiguration
	hasVar bool
}

// Type 组件类型
func (x *GitCreateTagNode) Type() string {
	return "ci/gitCreateTag"
}

func (x *GitCreateTagNode) New() types.Node {
	return &GitCreateTagNode{Config: GitCreateTagNodeConfiguration{}}
}

// Init 初始化
func (x *GitCreateTagNode) Init(ruleConfig types.Config, configuration types.Configuration) error {
	err := maps.Map2Struct(configuration, &x.Config)
	err = maps.Map2Struct(configuration, &x.baseGitNode.Config)
	if str.CheckHasVar(x.Config.Directory) || str.CheckHasVar(x.Config.Tag) || str.CheckHasVar(x.Config.Signature.AuthorName) || str.CheckHasVar(x.Config.Signature.AuthorEmail) {
		x.hasVar = true
	}
	return err
}

// OnMsg 处理消息
func (x *GitCreateTagNode) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
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
	commit, err := r.Head()
	if err != nil {
		// 处理错误
	}

	// 获取提交对象
	commitObj, err := r.CommitObject(commit.Hash())
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}

	tagger := object.Signature{
		Name:  x.getSignatureName(msg, evn),
		Email: x.getSignatureEmail(msg, evn),
		When:  time.Now(),
	}
	opts := &git.CreateTagOptions{
		Tagger:  &tagger,
		Message: x.getMessage(msg, evn),
	}
	// 创建附注标签
	annotatedTag, err := r.CreateTag(x.getTag(msg, evn), commitObj.Hash, opts)
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	msg.Metadata.PutValue(KeyHash, annotatedTag.Hash().String())
	ctx.TellSuccess(msg)
}

// Destroy 销毁
func (x *GitCreateTagNode) Destroy() {
}

func (x *GitCreateTagNode) getTag(_ types.RuleMsg, evn map[string]interface{}) string {
	tag := x.Config.Tag
	if evn != nil {
		tag = str.ExecuteTemplate(tag, evn)
	}
	return tag
}

func (x *GitCreateTagNode) getMessage(_ types.RuleMsg, evn map[string]interface{}) string {
	message := x.Config.Message
	if evn != nil {
		message = str.ExecuteTemplate(message, evn)
	}
	return message
}

func (x *GitCreateTagNode) getSignatureName(_ types.RuleMsg, evn map[string]interface{}) string {
	name := x.Config.Signature.AuthorName
	if evn != nil {
		name = str.ExecuteTemplate(name, evn)
	}
	return name
}

func (x *GitCreateTagNode) getSignatureEmail(_ types.RuleMsg, evn map[string]interface{}) string {
	email := x.Config.Signature.AuthorEmail
	if evn != nil {
		email = str.ExecuteTemplate(email, evn)
	}
	return email
}
