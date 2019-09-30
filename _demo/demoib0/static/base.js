
/* theme switching functionality */

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
	var styles = document.querySelectorAll('[rel="stylesheet"]');
	for (var i = 0; i < styles.length; i++) {
		if (styles[i].dataset.theme == n) {
			switchToTheme(styles[i]);
			return;
		}
	}
}

var currTheme = window.localStorage.getItem('site-theme-name');
if (currTheme)
	switchToThemeName(currTheme);
