package ui

import (
	"github.com/rivo/tview"
)

type AuthModal struct {
	pages *tview.Pages
	app   *tview.Application

	phoneForm    *tview.Form
	codeForm     *tview.Form
	passwordForm *tview.Form

	onPhone    func(phone string)
	onCode     func(code string)
	onPassword func(password string)
}

func NewAuthModal(app *tview.Application, pages *tview.Pages) *AuthModal {
	am := &AuthModal{
		pages: pages,
		app:   app,
	}
	am.buildForms()
	return am
}

func (am *AuthModal) buildForms() {
	// Phone form
	am.phoneForm = tview.NewForm().
		AddInputField("Phone Number", "", 20, nil, nil).
		AddButton("Submit", func() {
			phone := am.phoneForm.GetFormItemByLabel("Phone Number").(*tview.InputField).GetText()
			if phone != "" && am.onPhone != nil {
				am.onPhone(phone)
				am.pages.HidePage("auth")
			}
		})
	am.phoneForm.SetBorder(true).SetTitle(" Enter Phone Number ")

	// Code form
	am.codeForm = tview.NewForm().
		AddInputField("Verification Code", "", 10, nil, nil).
		AddButton("Submit", func() {
			code := am.codeForm.GetFormItemByLabel("Verification Code").(*tview.InputField).GetText()
			if code != "" && am.onCode != nil {
				am.onCode(code)
				am.pages.HidePage("auth")
			}
		})
	am.codeForm.SetBorder(true).SetTitle(" Enter Verification Code ")

	// Password form
	am.passwordForm = tview.NewForm().
		AddPasswordField("2FA Password", "", 20, '*', nil).
		AddButton("Submit", func() {
			pw := am.passwordForm.GetFormItemByLabel("2FA Password").(*tview.InputField).GetText()
			if pw != "" && am.onPassword != nil {
				am.onPassword(pw)
				am.pages.HidePage("auth")
			}
		})
	am.passwordForm.SetBorder(true).SetTitle(" Enter 2FA Password ")
}

func (am *AuthModal) SetCallbacks(onPhone func(string), onCode func(string), onPassword func(string)) {
	am.onPhone = onPhone
	am.onCode = onCode
	am.onPassword = onPassword
}

func (am *AuthModal) ShowPhone() {
	am.phoneForm.GetFormItemByLabel("Phone Number").(*tview.InputField).SetText("")
	centered := am.center(am.phoneForm, 50, 7)
	am.pages.AddAndSwitchToPage("auth", centered, true)
	am.app.SetFocus(am.phoneForm)
}

func (am *AuthModal) ShowCode() {
	am.codeForm.GetFormItemByLabel("Verification Code").(*tview.InputField).SetText("")
	centered := am.center(am.codeForm, 50, 7)
	am.pages.AddAndSwitchToPage("auth", centered, true)
	am.app.SetFocus(am.codeForm)
}

func (am *AuthModal) ShowPassword() {
	am.passwordForm.GetFormItemByLabel("2FA Password").(*tview.InputField).SetText("")
	centered := am.center(am.passwordForm, 50, 7)
	am.pages.AddAndSwitchToPage("auth", centered, true)
	am.app.SetFocus(am.passwordForm)
}

func (am *AuthModal) center(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewGrid().
		SetColumns(0, width, 0).
		SetRows(0, height, 0).
		AddItem(p, 1, 1, 1, 1, 0, 0, true)
}
