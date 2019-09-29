
function switchToTheme(s) {
	var e = document.getElementById("theme");
	if (e) {
		//e.href = s.href;

		//// XXX non-default may be active one
		//e.disabled = true;
		//s.disabled = false;

		//e.rel = 'alternate stylesheet';
		//s.rel = 'stylesheet';

		e.media = "none";
		s.media = "";
	}
}

function switchToThemeName(n) {
	var alts = document.querySelectorAll('[rel="stylesheet"]');
	for (var i = 0; i < alts.length; i++) {
		if (alts[i].dataset.theme == n) {
			switchToTheme(alts[i]);
			break;
		}
	}
}

var currTheme = window.localStorage.getItem('site-theme-name');
if (currTheme)
	switchToThemeName(currTheme);
