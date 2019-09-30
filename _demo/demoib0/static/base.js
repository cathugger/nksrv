
var theme_default = document.getElementById("theme");
var theme_current = theme_default;

function switchToTheme(theme_new) {
	if (theme_current) {
		theme_current.media = "none";
		theme_new.media = "";
		theme_current = theme_new;
	}
}

function switchToThemeName(n) {
	if (n == "") {
		switchToTheme(theme_default);
		return;
	}
	var alts = document.querySelectorAll('[rel="stylesheet"]');
	for (var i = 0; i < alts.length; i++) {
		if (alts[i].dataset.theme == n) {
			switchToTheme(alts[i]);
			return;
		}
	}
}

var currTheme = window.localStorage.getItem('site-theme-name');
if (currTheme)
	switchToThemeName(currTheme);
