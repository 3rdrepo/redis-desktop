package main

import (
	"encoding/json"
	"fmt"
	"github.com/lxn/win"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/chenqinghe/walk"
	. "github.com/chenqinghe/walk/declarative"
	"github.com/sirupsen/logrus"
)

type Directory struct {
	parent   *Directory
	Name     string
	Children []walk.TreeItem
}

func NewDirectory(name string, parent *Directory) *Directory {
	return &Directory{Name: name, parent: parent}
}

func (d *Directory) Text() string {
	return d.Name
}

func (d *Directory) Parent() walk.TreeItem {
	if d.parent == nil {
		return nil
	}
	return d.parent
}

func (d *Directory) ChildCount() int {
	return len(d.Children)
}

func (d *Directory) ChildAt(i int) walk.TreeItem {
	return d.Children[i]
}

func (d *Directory) Image() interface{} {
	return "img/dir.ico"
}

type Session struct {
	Key      string
	Host     string
	Port     int
	Password string
	parent   *Directory
}

func (s *Session) Image() interface{} {
	return "img/redis.ico"
}

func (s *Session) Text() string {
	if s.Key != "" {
		return s.Key
	}
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

func (s *Session) Parent() walk.TreeItem {
	if s.parent == nil {
		return nil
	}
	return s.parent
}

func (s *Session) ChildCount() int {
	return 0
}

func (s *Session) ChildAt(i int) walk.TreeItem {
	return nil
}

var _ walk.TreeItem = new(Directory)
var _ walk.TreeItem = new(Session)

type SessionTreeModel struct {
	walk.TreeModelBase
	roots []walk.TreeItem
}

func (m *SessionTreeModel) RootCount() int {
	return len(m.roots)
}

func (m *SessionTreeModel) RootAt(index int) walk.TreeItem {
	return m.roots[index]
}

func NewSessionTreeModel() *SessionTreeModel {
	m := &SessionTreeModel{
		TreeModelBase: walk.TreeModelBase{},
		roots:         []walk.TreeItem{},
	}

	return m
}

type TreeViewEx struct {
	*walk.TreeView

	root  *MainWindowEX
	model *SessionTreeModel
}

type Facade struct {
	// Directory
	Name     string
	Children []*Facade

	// Session
	Host     string
	Port     int
	Password string
}

func (f Facade) IsDirectory() bool {
	return f.Name != ""
}

func (tv *TreeViewEx) LoadSession(data []byte) error {
	facades := make([]Facade, 0)
	if err := json.Unmarshal(data, &facades); err != nil {
		return err
	}

	for _, v := range facades {
		tv.model.roots = append(tv.model.roots, buildModel(nil, v))
	}

	return tv.SetModel(tv.model)
}

func (tv *TreeViewEx) SaveSession(file string) error {
	data, err := json.Marshal(tv.model.roots)
	if err != nil {
		return err
	}

RETRY:
	if err := ioutil.WriteFile(file, data, os.ModePerm); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(file), os.ModePerm); err != nil {
				return err
			}
			goto RETRY
		}
		return err
	}
	return nil
}

func buildModel(parent *Directory, facade Facade) walk.TreeItem {
	if !facade.IsDirectory() {
		return &Session{
			Host:     facade.Host,
			Port:     facade.Port,
			Password: facade.Password,
			parent:   parent,
		}
	}
	dir := &Directory{
		parent: parent,
		Name:   facade.Name,
	}
	for _, v := range facade.Children {
		dir.Children = append(dir.Children, buildModel(dir, *v))
	}

	return dir
}

func (tv *TreeViewEx) AddSession() {
	var (
		s Session

		dlg      *walk.Dialog
		accepted bool

		widgetName     *walk.LineEdit
		widgetHost     *walk.LineEdit
		widgetPort     *walk.LineEdit
		widgetPassword *walk.LineEdit
	)

	var itemSelected bool
	if tv.CurrentItem() != nil {
		itemSelected = true
	}
	logrus.Debugln("before create session, itemSelected:", itemSelected)

	if _, err := (Dialog{
		Title:     "新建会话",
		AssignTo:  &dlg,
		Size:      Size{400, 550},
		FixedSize: true,
		Layout:    VBox{MarginsZero: true},
		Children: []Widget{
			Composite{
				Layout: HBox{Margins: Margins{Top: 20, Left: 20, Right: 20}},
				Children: []Widget{
					TextLabel{Text: "Name:", MaxSize: Size{50, 0}, MinSize: Size{50, 0}, TextAlignment: AlignHNearVCenter},
					LineEdit{AssignTo: &widgetName, MinSize: Size{150, 0}, MaxSize: Size{150, 0}},
				},
			},
			Composite{
				Layout: HBox{Margins: Margins{Left: 20, Right: 20}},
				Children: []Widget{
					TextLabel{Text: "Host:", MaxSize: Size{50, 0}, MinSize: Size{50, 0}, TextAlignment: AlignHNearVCenter},
					LineEdit{AssignTo: &widgetHost, MinSize: Size{150, 0}, MaxSize: Size{150, 0}},
				},
			},
			Composite{
				Layout: HBox{Margins: Margins{Left: 20, Right: 20}},
				Children: []Widget{
					TextLabel{Text: "Port:", MaxSize: Size{50, 0}, MinSize: Size{50, 0}, TextAlignment: AlignHNearVCenter},
					LineEdit{AssignTo: &widgetPort, MinSize: Size{150, 0}, MaxSize: Size{150, 0}},
				},
			},
			Composite{
				Layout: HBox{Margins: Margins{Left: 20, Right: 20, Bottom: 30}},
				Children: []Widget{
					TextLabel{Text: "Password:", MaxSize: Size{50, 0}, MinSize: Size{50, 0}, TextAlignment: AlignHNearVCenter},
					LineEdit{AssignTo: &widgetPassword, MinSize: Size{150, 0}, MaxSize: Size{150, 0}},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{
						MaxSize: Size{100, 30},
						Text:    "确定",
						OnClicked: func() {
							s.Key = widgetName.Text()
							s.Host = widgetHost.Text()
							s.Password = widgetPassword.Text()
							portStr := widgetPort.Text()
							p, err := strconv.Atoi(portStr)
							if err != nil {
								walk.MsgBox(dlg, "ERROR", "invalid port", walk.MsgBoxIconError)
								return
							}
							s.Port = p
							accepted = true
							dlg.Close(0)
						},
					},
					PushButton{
						MaxSize: Size{100, 30},
						Text:    "取消",
						OnClicked: func() {
							dlg.Close(0)
						},
					},
				},
			},
		},
	}).Run(tv.root); err != nil {
		logrus.Errorln("run new session dialog error:", err)
	}

	if !accepted {
		return
	}

	logrus.Debugln("after create session, itemSelected:", itemSelected)
	if !itemSelected {
		tv.SetCurrentItem(nil)
	}

	logrus.Debugln("add session")
	tv.addSession(&s)
	logrus.Debugln("reload model")
	tv.ReloadModel()
	tv.EnsureVisible(&s)
	//if err := tv.SaveSession(tv.root.sessionFile); err != nil {
	//	logrus.Errorln("save sessions error:", err)
	//}
}

func (tv *TreeViewEx) addSession(s *Session) {
	item := tv.CurrentItem()
	if item == nil {
		tv.model.roots = append(tv.model.roots, s)
		return
	}

	switch t := item.(type) {
	case *Directory:
		s.parent = t
		t.Children = append(t.Children, s)
	case *Session:
		// TODO: 未选择任何item的情况下新建session，关闭新建session对话框后，会默认选择第一个TreeItem，
		// 造成item.(*Directory)断言失败，因此这里还是需要判断选中session的情况。
		if t.parent == nil { // root session
			tv.model.roots = append(tv.model.roots, s)
			return
		}
		dir := t.parent
		s.parent = dir
		dir.Children = append(dir.Children, s)
	}
}

func (tv *TreeViewEx) AddSessions(sesses []Session) {
	//for _, sess := range sesses {
	//	tv.addSession(sess)
	//}
	//tv.ReloadModel()
	//tv.root.saveSessions(tv.sessions)
}

func (tv *TreeViewEx) GetSessions() []Session {
	return nil
	//return tv.sessions
}

func (tv *TreeViewEx) RemoveSelectedSession() {
	s := tv.CurrentItem().(*Session)
	p := s.parent
	if p == nil { // root session
		for k, v := range tv.model.roots {
			if v == s {
				copy(tv.model.roots[k:], tv.model.roots[k+1:])
				tv.model.roots = tv.model.roots[:len(tv.model.roots)-1]
				break
			}
		}
		tv.ReloadModel()
		return
	}

	for k, v := range p.Children {
		if v == s {
			copy(p.Children[k:], p.Children[k+1:])
			p.Children = p.Children[:len(p.Children)-1]
			break
		}
	}

	tv.ReloadModel()
	return
}

func (tv *TreeViewEx) RemoveSelectedDirectory() {
	s := tv.CurrentItem().(*Directory)
	if len(s.Children) > 0 {
		key := walk.MsgBox(tv.root, "Confirm Remove?", "the directory not empty, are you sure to remove?", walk.MsgBoxIconQuestion|walk.MsgBoxYesNo)
		if key != win.IDYES {
			return
		}
	}

	p := s.parent
	if p == nil { // root session
		for k, v := range tv.model.roots {
			if v == s {
				copy(tv.model.roots[k:], tv.model.roots[k+1:])
				tv.model.roots = tv.model.roots[:len(tv.model.roots)-1]
				break
			}
		}
		tv.ReloadModel()
		return
	}

	for k, v := range p.Children {
		if v == s {
			copy(p.Children[k:], p.Children[k+1:])
			p.Children = p.Children[:len(p.Children)-1]
			break
		}
	}

	tv.ReloadModel()
	return
}

func (tv *TreeViewEx) ReloadModel() {
	tv.SetModel(tv.model)
}

func (tv *TreeViewEx) AddDirectory() {
	var parent *Directory
	var item walk.TreeItem

	curItem := tv.CurrentItem()
	switch t := curItem.(type) {
	case *Directory:
		parent = t
	case *Session:
		walk.MsgBox(tv.root, "ERROR", "不能在会话中创建目录", walk.MsgBoxIconError)
		return
	default: // nil
		name := (&SimpleDialog{}).Prompt(tv.root, "请输入目录名称")
		if name == "" {
			return
		}
		item = &Directory{
			parent:   nil,
			Name:     name,
			Children: nil,
		}
		tv.model.roots = append(tv.model.roots, item)
		tv.ReloadModel()
		tv.EnsureVisible(item)
		return
	}

	name := (&SimpleDialog{}).Prompt(tv.root, "请输入目录名称")
	if name == "" {
		return
	}
	item = &Directory{
		parent:   parent,
		Name:     name,
		Children: nil,
	}
	parent.Children = append(parent.Children, item)

	tv.ReloadModel()
	tv.EnsureVisible(item)
}