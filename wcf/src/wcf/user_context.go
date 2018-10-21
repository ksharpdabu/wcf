package wcf

import "sync"

type UserContext struct {
    Limit Limiter
	Info *UserInfo
}

var allctx = make(map[string]* UserContext)
var mu sync.Mutex

func GetOrCreateContext(info *UserInfo) *UserContext {
	var ctx *UserContext
	var exist bool
	if ctx, exist = allctx[info.User]; !exist {
		mu.Lock()
		if ctx, exist = allctx[info.User]; !exist {
			ctx = BuildFromUserInfo(info)
			allctx[info.User] = ctx
		}
		mu.Unlock()
	}
	return ctx
}

func BuildFromUserInfo(info *UserInfo) *UserContext {
	ctx := &UserContext{}
	ctx.Limit.Reset(info.MaxConnection, info.MaxConnection)
	ctx.Info = info
	return ctx
}