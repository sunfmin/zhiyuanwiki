package core

import (
	"regexp"
	"strings"

	"github.com/xuri/excelize/v2"
)

var parenRe = regexp.MustCompile(`[(（].*?[)）]`)

// NormName 归一化院校/专业名：全角括号/空格→半角、去空白。
func NormName(s string) string {
	return strings.NewReplacer("（", "(", "）", ")", "　", "", " ", "").Replace(strings.TrimSpace(s))
}

// BaseName 去掉括号注记（分校/校区），用于母体标签继承。
func BaseName(s string) string {
	return strings.TrimSpace(parenRe.ReplaceAllString(NormName(s), ""))
}

// 12 学科门类 → 1 字码。其他归「他」。客户端 src/lib/menlei.ts / filters 有镜像，改这里要同步。
var menleiToCode = map[string]string{
	"哲学": "哲", "经济学": "经", "法学": "法", "教育学": "教",
	"文学": "文", "历史学": "史", "理学": "理", "工学": "工",
	"农学": "农", "医学": "医", "管理学": "管", "艺术学": "艺",
}

// MenleiOther 是未命中任何门类时的兜底码。
const MenleiOther = "他"

// firstParenRe 切掉首个括号及其后全部内容（旧表「专业」常带括号注记的子专业列表）。
var firstParenRe = regexp.MustCompile(`[(（].*$`)

// MajorBase 归一化专业名并切到首个括号前，作为跨表挂接的稳定基名。
func MajorBase(s string) string {
	return strings.TrimSpace(firstParenRe.ReplaceAllString(NormName(s), ""))
}

// MenleiClassifier 把专业名归到 12 学科门类之一（返回 1 字码），未命中返回「他」。
// 先用从带「门类」列的表学到的「专业名→门类」精确映射，再退到关键词启发式。
type MenleiClassifier struct {
	exact map[string]string // 归一化（含去括号基名）专业名 → 1 字码
}

func NewMenleiClassifier() *MenleiClassifier {
	return &MenleiClassifier{exact: map[string]string{}}
}

// learn 记录一条 专业名→门类（仅当门类是 12 门类之一），按全名与去括号基名各建一键。
func (mc *MenleiClassifier) learn(major, menlei string) {
	code := menleiToCode[strings.TrimSpace(menlei)]
	if code == "" || strings.TrimSpace(major) == "" {
		return
	}
	for _, k := range []string{NormName(major), MajorBase(major)} {
		if k != "" {
			if _, ok := mc.exact[k]; !ok {
				mc.exact[k] = code
			}
		}
	}
}

// Learn 公开一条 专业名→门类 学习入口，供从 DB 行（而非 xlsx）重建分类器。见 ADR-0014。
func (mc *MenleiClassifier) Learn(major, menlei string) { mc.learn(major, menlei) }

// Len 返回学到的精确条目数（用于日志）。
func (mc *MenleiClassifier) Len() int { return len(mc.exact) }

// 关键词启发式，按顺序匹配（更具体/更易混的在前）。仅作长尾兜底，精确映射优先。
var menleiKeywords = []struct {
	code string
	subs []string
}{
	// 农 先于 医：动物医学/动植物检疫属农学却含「医」。
	{"农", []string{"农学", "农业", "园艺", "园林", "林学", "水产", "动物", "植物", "草业", "茶学", "蜂学", "种子", "烟草", "兽医", "渔", "水土保持"}},
	{"医", []string{"医学", "临床", "口腔", "护理", "药学", "中药", "中医", "针灸", "推拿", "预防", "麻醉", "影像", "检验", "康复", "法医", "卫生", "眼视光", "助产", "医技", "药物", "药品"}},
	{"史", []string{"历史", "考古", "文物", "文化遗产"}},
	{"哲", []string{"哲学", "逻辑", "宗教", "伦理"}},
	{"艺", []string{"音乐", "美术", "设计", "表演", "戏剧", "影视", "电影", "舞蹈", "绘画", "雕塑", "摄影", "动画", "视觉", "播音", "主持", "书法", "艺术", "服装与服饰"}},
	{"教", []string{"教育", "师范", "学前", "体育", "运动", "武术"}},
	// 文/语 先于 法：「法语」含「语」属文学，「法学/法律」才是法学。
	{"文", []string{"汉语", "语言", "文学", "新闻", "传播", "广告", "编辑", "出版", "翻译", "英语", "日语", "德语", "法语", "俄语", "西班牙语", "朝鲜语", "阿拉伯语", "葡萄牙语", "意大利语", "新媒体", "秘书"}},
	{"法", []string{"法学", "法律", "知识产权", "政治", "社会学", "社会工作", "公安", "侦查", "治安", "思想政治", "马克思", "民族", "外交", "监狱", "警", "海关"}},
	{"经", []string{"经济", "金融", "财政", "税收", "税务", "贸易", "保险", "投资"}},
	{"管", []string{"管理", "会计", "财务", "工商", "营销", "人力资源", "物流", "电子商务", "审计", "图书", "档案", "酒店", "物业", "房地产", "资产评估"}},
	// 工 与 理 多有交叠；工（含「工程/技术」）放在理前作兜底。
	{"工", []string{"工程", "技术", "机械", "电气", "电子", "计算机", "软件", "自动化", "通信", "土木", "建筑", "材料", "能源", "动力", "化工", "环境", "航空", "航天", "车辆", "测绘", "网络", "物联网", "智能", "机器人", "采矿", "冶金", "纺织", "食品", "船舶", "兵器", "光电", "焊接", "印刷", "包装", "石油", "矿", "给排水", "供热", "机电", "数据科学", "信息安全", "遥感", "海洋工程", "安全"}},
	{"理", []string{"数学", "物理", "化学", "生物", "天文", "地理", "地质", "海洋", "大气", "统计", "心理", "力学", "信息与计算"}},
}

// Code 把专业名归到 1 字门类码：精确映射 → 关键词启发式 → 「他」。
func (mc *MenleiClassifier) Code(name string) string {
	if c, ok := mc.exact[NormName(name)]; ok {
		return c
	}
	if c, ok := mc.exact[MajorBase(name)]; ok {
		return c
	}
	n := NormName(name)
	for _, k := range menleiKeywords {
		for _, s := range k.subs {
			if strings.Contains(n, s) {
				return k.code
			}
		}
	}
	return MenleiOther
}

// addRows 从一张带「门类」+「专业(名称)」列的表学习精确映射（表头在前 4 行内）。
// 无相关列的表静默跳过。万师兄旧科类表与浙江一表联动均适用。
func (mc *MenleiClassifier) addRows(rows [][]string) {
	headerIdx := -1
	for i := 0; i < len(rows) && i < 4; i++ {
		if HasCell(rows[i], "门类") && (HasCell(rows[i], "专业") || HasCell(rows[i], "专业名称")) {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 {
		return
	}
	h := rows[headerIdx]
	cMenlei := FindCol(h, "门类")
	cMajor := FindCol(h, "专业", "专业名称")
	if cMenlei < 0 || cMajor < 0 {
		return
	}
	for _, r := range rows[headerIdx+1:] {
		mc.learn(Cell(r, cMajor), Cell(r, cMenlei))
	}
}

// LoadMenlei 从多个带「门类」+「专业」列的 xlsx 学习 专业名→门类 精确映射。
// 打不开/无该列的文件静默跳过。
func LoadMenlei(paths []string) *MenleiClassifier {
	mc := NewMenleiClassifier()
	for _, p := range paths {
		f, err := excelize.OpenFile(p)
		if err != nil {
			continue
		}
		if sheets := f.GetSheetList(); len(sheets) > 0 {
			if rows, err := f.GetRows(sheets[0]); err == nil {
				mc.addRows(rows)
			}
		}
		_ = f.Close()
	}
	return mc
}
