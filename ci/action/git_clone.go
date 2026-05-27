package action

import (
	"crypto/tls"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/client"
	httptransport "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/maps"
	"github.com/rulego/rulego/utils/str"
	"net/http"
	"os"
)

func init() {
	_ = rulego.Registry.Register(&GitCloneNode{})

	//不验证https
	var c = httptransport.NewClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	})
	client.InstallProtocol("https", c)
}

// KeyWorkDir 工作目录
const KeyWorkDir = "workDir"

// KeyRef 分支
const KeyRef = "ref"

// KeyGitSshUrl 仓库Ssh地址
const KeyGitSshUrl = "gitSshUrl"

// KeyGitHttpUrl 仓库Http地址
const KeyGitHttpUrl = "gitHttpUrl"

// GitCloneNodeConfiguration 节点配置
type GitCloneNodeConfiguration struct {
	// Git 仓库 URL
	Repository string `json:"repository" label:"Repository" desc:"Git repository URL"`
	// 克隆到的本地目录
	Directory string `json:"directory" label:"Directory" desc:"Local directory to clone into"`
	// 分支或标签的完整引用名
	Reference string `json:"reference" label:"Reference" desc:"Branch or tag reference, e.g. refs/heads/main"`
	// 认证类型，可以是 "ssh", "password", 或 "token"
	AuthType string `json:"authType" label:"Auth Type" desc:"Authentication type: ssh, password, token"`
	// 用户名
	AuthUser string `json:"authUser" label:"Auth User" desc:"Authentication username"`
	// 密码或 token
	AuthPassword string `json:"authPassword" label:"Auth Password" desc:"Authentication password or token"`
	// SSH 秘钥文件路径
	AuthPemFile string `json:"authPemFile" label:"Auth PEM File" desc:"SSH private key file path"`
	// 代理地址
	ProxyUrl string `json:"proxyUrl" label:"Proxy URL" desc:"Proxy server URL"`
	// 代理用户名
	ProxyUsername string `json:"proxyUsername" label:"Proxy Username" desc:"Proxy authentication username"`
	// 代理密码
	ProxyPassword string `json:"proxyPassword" label:"Proxy Password" desc:"Proxy authentication password"`
}

// GitCloneNode 实现 Git 仓库克隆
type GitCloneNode struct {
	baseGitNode
	// 节点配置
	Config GitCloneNodeConfiguration
	hasVar bool
}

// Type 组件类型
func (x *GitCloneNode) Type() string {
	return "ci/gitClone"
}

func (x *GitCloneNode) New() types.Node {
	return &GitCloneNode{Config: GitCloneNodeConfiguration{
		Repository:   "",
		Directory:    "",
		AuthType:     "token",
		AuthPassword: "${vars.token}",
		Reference:    "refs/heads/main",
	}}
}

// Init 初始化
func (x *GitCloneNode) Init(ruleConfig types.Config, configuration types.Configuration) error {
	err := maps.Map2Struct(configuration, &x.Config)
	err = maps.Map2Struct(configuration, &x.baseGitNode.Config)
	if str.CheckHasVar(x.Config.Repository) || str.CheckHasVar(x.Config.Directory) || str.CheckHasVar(x.Config.Reference) {
		x.hasVar = true
	}
	return err
}

// OnMsg 处理消息
func (x *GitCloneNode) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	var evn map[string]interface{}
	if x.hasVar {
		evn = base.NodeUtils.GetEvnAndMetadata(ctx, msg)
	}
	ref := x.getReferenceName(msg, evn)
	workDir := x.getWorkDir(msg, evn)
	msg.Metadata.PutValue(KeyWorkDir, workDir)
	repository := x.getRepository(msg, evn)
	// 检查目录是否存在
	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		// 设置克隆选项
		cloneOptions := &git.CloneOptions{
			URL:      repository,
			Progress: os.Stdout,
		}
		if proxy := x.getProxy(); proxy.URL != "" {
			cloneOptions.ProxyOptions = proxy
		}
		// 如果指定了分支或标签，则设置为克隆特定的引用
		if ref != "" {
			cloneOptions.ReferenceName = plumbing.ReferenceName(ref)
		}

		// 根据 AuthType 字段的值选择认证方式
		if auth, err := x.getAuthMethod(); err != nil {
			ctx.TellFailure(msg, err)
			return
		} else {
			cloneOptions.Auth = auth
		}
		// 执行克隆操作
		if _, err := git.PlainClone(workDir, false, cloneOptions); err != nil {
			ctx.TellFailure(msg, err)
		} else {
			ctx.TellSuccess(msg)
		}
	} else {
		// 目录存在，执行拉取操作
		r, err := git.PlainOpen(workDir)
		if err != nil {
			ctx.TellFailure(msg, err)
			return
		}
		w, err := r.Worktree()
		if err != nil {
			ctx.TellFailure(msg, err)
			return
		}
		pullOptions := &git.PullOptions{
			//RemoteName: "origin",
			RemoteURL: repository,
			Force:     true,
		}
		if proxy := x.getProxy(); proxy.URL != "" {
			pullOptions.ProxyOptions = proxy
		}
		if ref != "" {
			pullOptions.ReferenceName = plumbing.ReferenceName(ref)
		}
		// 根据 AuthType 字段的值选择认证方式
		if auth, err := x.getAuthMethod(); err != nil {
			ctx.TellFailure(msg, err)
			return
		} else {
			pullOptions.Auth = auth
		}
		if err = w.Pull(pullOptions); err != nil {
			if err == git.NoErrAlreadyUpToDate {
				ctx.TellSuccess(msg)
			} else {
				ctx.TellFailure(msg, err)
			}
		} else {
			ctx.TellSuccess(msg)
		}
	}
}

// Destroy 销毁
func (x *GitCloneNode) Destroy() {
}

// Desc returns the component description
func (x *GitCloneNode) Desc() string {
	return "Git clone repository to local directory. Supports ssh, password, token auth. Routes to Success/Failure"
}
