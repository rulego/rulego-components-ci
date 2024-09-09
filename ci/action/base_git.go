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
	"crypto/tls"
	"errors"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/client"
	httptransport "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/utils/str"
	"net/http"
	"path"
	"strings"
)

func init() {
	//不验证https
	var c = httptransport.NewClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	})
	client.InstallProtocol("https", c)
}

type baseGitNodeConfiguration struct {
	// Git 仓库 URL
	Repository string
	// 克隆到的本地目录
	Directory string
	// 分支或标签的完整引用名
	Reference string
	// 认证类型，可以是 "ssh", "password", 或 "token"
	AuthType string
	// 用户名
	AuthUser string
	// 密码或 token
	AuthPassword string
	// SSH 秘钥文件路径
	AuthPemFile string
	// 代理地址
	ProxyUrl string
	// 代理用户名
	ProxyUsername string
	// 代理密码
	ProxyPassword string
	//RefSpecs 用于定义本地分支与远程分支之间的映射关系，多个映射关系与逗号隔开，例如：refs/heads/your-branch:refs/heads/your-branch
	RefSpecs string
}

type baseGitNode struct {
	Config baseGitNodeConfiguration
}

func (x *baseGitNode) getAuthMethod() (transport.AuthMethod, error) {
	// 根据 AuthType 字段的值选择认证方式
	switch x.Config.AuthType {
	case "ssh-key", "ssh":
		// 使用 SSH 秘钥文件
		sshKey, err := ssh.NewPublicKeysFromFile(x.Config.AuthUser, x.Config.AuthPemFile, x.Config.AuthPassword)
		if err != nil {
			return nil, err
		}
		return sshKey, nil
	case "username-password", "password":
		// 使用用户名和密码
		auth := &httptransport.BasicAuth{
			Username: x.Config.AuthUser,
			Password: x.Config.AuthPassword,
		}
		return auth, nil
	case "token":
		// 使用 token
		auth := &httptransport.BasicAuth{
			Username: x.Config.AuthUser, // 注意：GitHub 个人访问令牌使用时，用户名可以是任意字符串
			Password: x.Config.AuthPassword,
		}
		return auth, nil
	}
	return nil, errors.New("not authType=" + x.Config.AuthType)
}

func (x *baseGitNode) getWorkDir(msg types.RuleMsg, evn map[string]interface{}) string {
	workDir := x.Config.Directory
	if workDir == "" {
		workDir = msg.Metadata.GetValue(KeyWorkDir)
	} else if evn != nil {
		workDir = str.ExecuteTemplate(workDir, evn)
	}
	workDir = path.Join(workDir, x.getRepoName(x.getRepository(msg, evn)))
	return workDir
}

func (x *baseGitNode) getRefSpecs(msg types.RuleMsg, evn map[string]interface{}) []config.RefSpec {
	ref := x.Config.RefSpecs
	if evn != nil {
		ref = str.ExecuteTemplate(ref, evn)
	}
	values := strings.Split(ref, ",")
	var refSpecs []config.RefSpec
	for _, item := range values {
		refSpecs = append(refSpecs, config.RefSpec(item))
	}
	return refSpecs
}

func (x *baseGitNode) getRepository(msg types.RuleMsg, evn map[string]interface{}) string {
	repository := x.Config.Repository
	if repository == "" {
		if x.Config.AuthType == "ssh-key" || x.Config.AuthType == "ssh" {
			repository = msg.Metadata.GetValue(KeyGitSshUrl)
		} else {
			repository = msg.Metadata.GetValue(KeyGitHttpUrl)
		}
	} else if evn != nil {
		repository = str.ExecuteTemplate(repository, evn)
	}
	return repository
}

func (x *baseGitNode) getReferenceName(msg types.RuleMsg, evn map[string]interface{}) string {
	ref := x.Config.Reference
	if ref == "" {
		ref = msg.Metadata.GetValue(KeyRef)
	} else if evn != nil {
		ref = str.ExecuteTemplate(ref, evn)
	}
	return ref
}

// GetRepoName 从 Git 仓库 URL 中提取仓库名称
func (x *baseGitNode) getRepoName(repoURL string) string {
	// 分割 URL 来获取仓库名称部分
	parts := strings.Split(repoURL, "/")
	// 仓库名称是 URL 的最后一部分
	repoName := parts[len(parts)-1]
	// 移除 ".git" 后缀
	repoName = strings.TrimSuffix(repoName, ".git")
	return repoName
}

func (x *baseGitNode) getProxy() transport.ProxyOptions {
	if x.Config.ProxyUrl != "" {
		return transport.ProxyOptions{
			URL:      x.Config.ProxyUrl,
			Username: x.Config.ProxyUsername,
			Password: x.Config.ProxyPassword,
		}
	}
	return transport.ProxyOptions{}
}
