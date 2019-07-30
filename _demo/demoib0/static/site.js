// console.log("ohayo!");

function expandimg(lnk, thm) {
	var exp;
	var exps = lnk.getElementsByClassName("imgexp");
	if (exps.length > 0) {
		// expanded img element already exists - reuse
		exp = exps[0];
		// before un-hiding, remove from DOM
		lnk.removeChild(exp);
		// un-hide
		exp.style.removeProperty('display');
	} else {
		// make new expanded img element
		exp = new Image();
		exp.src = lnk.href;
		exp.width  = lnk.dataset.width;
		exp.height = lnk.dataset.height;
		exp.className = 'imgexp';
	}
	// swap expanded image with thumbnail
	lnk.replaceChild(exp, thm);
	// now thm is out of DOM, hide it and put it back
	thm.style.display = 'none';
	lnk.insertBefore(thm, exp);
}

function unexpandimg(lnk, thm, exp) {
	// before un-hiding, remove from DOM
	lnk.removeChild(thm);
	// un-hide
	thm.style.removeProperty('display');
	// swap
	lnk.replaceChild(thm, exp);
	// hide exp
	exp.style.display = 'none';
	// add hidden
	lnk.appendChild(exp);

	// (attempt to) fix scroll position
	// current position from top
	var currpos = document.documentElement.scrollTop || document.body.scrollTop;
	// console.log("currpos: " + currpos);
	// current element top RELATIVE TO currpos, minus some additional stuff to feel more natural
	var filetop = lnk.parentElement.getBoundingClientRect().top - 18;
	// if we're beyond thumbnail image
	// NOTE: NOT whole post but just specific thumbnail
	if (filetop < 0) {
		// scroll to it
		var newpos = currpos + filetop;
		document.documentElement.scrollTop = newpos;
		document.body.scrollTop            = newpos;
	}
}

function onglobalclick(e) {
	var tgt = e.target;
	if (tgt.className == "imgthumb") {
		var lnk = tgt.parentElement;
		var typ = lnk.dataset.type;
		//console.log(">image thumb clicked, type=" + typ);
		if (typ == 'image') {
			// perform atomic swap of expanded img and thumbnail
			expandimg(lnk, tgt);

			// don't actually open link
			e.preventDefault();
		}
	} else if (tgt.className == "imgexp") {
		// if we encounter this type then we already know that this is image
		var lnk = tgt.parentElement;
		var thm = lnk.getElementsByClassName('imgthumb')[0];
		unexpandimg(lnk, thm, tgt);

		// don't actually open link
		e.preventDefault();
	}
}

document.documentElement.addEventListener("click", onglobalclick);
