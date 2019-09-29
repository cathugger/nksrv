
function switchToTheme(s) {
	var e = document.getElementById("theme");
	if (e)
		e.href = s.href;
}

function switchToThemeName(n) {
	var alts = document.querySelectorAll('[rel="alternate stylesheet"]');
	for (var i = 0; i < alts.length; i++) {
		if (alts[i].title == currTheme) {
			switchToTheme(alts[i]);
			break;
		}
	}
}

var currTheme = window.localStorage.getItem('site-theme-name');
if (currTheme)
	switchToThemeName(currTheme);
