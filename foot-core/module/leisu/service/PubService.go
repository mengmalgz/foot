package service

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"tesou.io/platform/foot-parent/foot-api/common/base"
	"tesou.io/platform/foot-parent/foot-api/module/analy/pojo"
	"tesou.io/platform/foot-parent/foot-core/module/analy/service"
	constants2 "tesou.io/platform/foot-parent/foot-core/module/leisu/constants"
	utils2 "tesou.io/platform/foot-parent/foot-core/module/leisu/utils"
	vo2 "tesou.io/platform/foot-parent/foot-core/module/leisu/vo"
	"time"
)

/**
发布推荐
*/
type PubService struct {
	service.AnalyService
	MatchPoolService
	PubLimitService
}

const content = "个人业余开发的一款足球分析程序，计算分析得出的结果.目前处于验证阶段，会持续同步发布出来，如发现近期命中率可以，可以选择跟进或参考，但一定要慎跟！！！！个人业余开发的一款足球分析程序，计算分析得出的结果.目前处于验证阶段，会持续同步发布出来，如发现近期命中率可以，可以选择跟进或参考，但一定要慎跟！！！"

/**
发布北京单场胜负过关
*/
func (this *PubService) PubBJDC() {
	option := 3
	//获取分析计算出的比赛列表
	analyList := this.AnalyService.GetPubDataList("Euro81_616Service", option)
	if len(analyList) <= 0 {
		base.Log.Info("1.当前无主队可发布的比赛!!!!")
		hours, _ := strconv.Atoi(time.Now().Format("15"))
		if hours < 24 && hours > 19 {
			//只在晚上处理
			base.Log.Info("1.1尝试获取客队可发布的比赛!!!!")
			option = 0
			analyList = this.AnalyService.GetPubDataList("Euro81_616Service", option)
			if len(analyList) <= 0 {
				base.Log.Info("1.2当前无客队可发布的比赛!!!!")
				return
			}
		}
	}

	//获取发布池的比赛列表
	matchPool := this.MatchPoolService.GetMatchList()
	//适配比赛,获取发布列表
	pubList := make(map[*pojo.AnalyResult]*vo2.MatchVO, 0)
	for _, analy := range analyList {
		analy_mainTeam := analy.MainTeamId
		for _, match := range matchPool {
			if strings.EqualFold(analy_mainTeam, match.MainTeam) {
				pubList[analy] = match
				break;
			}
		}
	}

	if len(pubList) <= 0 {
		base.Log.Info(fmt.Sprintf("2.当前无可发布的比赛!!!!比赛池size:%d,分析赛果size:%d", len(matchPool), len(analyList)))
		return
	}
	//打印要发布的比赛
	for _, match := range pubList {
		base.Log.Info("发布的比赛:", match.MatchDate, match.Numb, match.LeagueName, match.MainTeam, match.GuestTeam)
	}

	//发布比赛
	for analy, match := range pubList {
		//检查是否还有发布次数
		count := this.PubLimitService.HasPubCount()
		if !count {
			base.Log.Error("已经没有发布次数啦,请更换cookies帐号!")
			hours, _ := strconv.Atoi(time.Now().Format("15"))
			time.Sleep(time.Duration(int64(24-hours)) * time.Hour)
			break;
		}
		//检查是否是重复发布

		//----
		action := this.BJDCAction(match, option)
		if nil != action {
			switch action.Code {
			case 0, 100002:
				//0 成功 100002 每场比赛同一种玩法只可选择1次
				analy.LeisuPubd = true
				this.AnalyService.Modify(analy)
			case 100003:
				//100003 标题长度不正确
			default:
				break
			}

		}
		//需要间隔6分钟，再进行下一次发布
		intn := rand.Intn(10) + 6
		base.Log.Info("随机间隔时间为:", intn)
		time.Sleep(time.Duration(intn) * time.Minute)
	}
}

/**
发布比赛
*/
func (this *PubService) BJDCAction(param *vo2.MatchVO, option int) *vo2.PubRespVO {
	matchDate := param.MatchDate.Format("20060102150405")
	pubVO := new(vo2.PubVO)
	pubVO.Title = param.LeagueName + " " + matchDate + " " + param.MainTeam + "VS" + param.GuestTeam
	if len(pubVO.Title) < (3 * 15) {
		pubVO.Title = "足球精推:" + pubVO.Title
	}
	pubVO.Content = content
	pubVO.Multiple = 0
	pubVO.Price = 0
	//设置赔率
	oddData := param.GetBJDCOddData(option)
	if oddData == nil {
		//没有找到北单胜负过关的选项
		return nil;
	}
	infvo := vo2.MatchINFVO{}
	infvo.Id = param.DataId
	infvo.Selects = []int{oddData.DataSelects}
	infvo.Values = []float64{oddData.DataOdd}
	pubVO.Data = []vo2.MatchINFVO{infvo}
	//执行发布
	post := this.PubPost(pubVO)
	base.Log.Info("发布结果:" + pubVO.Title + " " + post.ToString())
	return post
}

/**
处理http post
*/
func (this *PubService) PubPost(param *vo2.PubVO) *vo2.PubRespVO {
	data := utils2.Post(constants2.PUB_URL, param)
	if len(data) <= 0 {
		base.Log.Error("PubPost:获取到的数据为空")
		return nil
	}
	base.Log.Info("http post 请求返回:" + data)
	resp := new(vo2.PubRespVO)
	json.Unmarshal([]byte(data), resp)
	return resp
}