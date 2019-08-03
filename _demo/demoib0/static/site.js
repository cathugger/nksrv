// console.log("ohayo!");

/* image expansion functionality */

function finishimgexpansion(lnk, exp, thm) {
	// mark as no longer loading
	delete thm.dataset.loadingexp;
	// remove expimg from DOM before modification
	lnk.removeChild(exp);
	// configure width and height
	exp.width  = lnk.dataset.width;
	exp.height = lnk.dataset.height;
	// un-hide
	exp.style.removeProperty('display');
	// perform replace thumb with expanded img
	lnk.replaceChild(exp, thm);
	// un-set loading indicator
	thm.style.removeProperty('opacity');
	// hide thumb
	thm.style.display = 'none';
	// reinject into DOM
	lnk.insertBefore(thm, exp);
}

function expandingimgerror(e) {
	var exp = e.target;
	var lnk = exp.parentElement;
	var thm = lnk.getElementsByClassName('imgthumb')[0];
	// mark as failed
	exp.dataset.failed = 1;
	// if it was waiting display
	if (thm.dataset.loadingexp) {
		// don't wait anymore
		clearTimeout(thm.dataset.loadingexp);
		// show it
		finishimgexpansion(lnk, exp, thm);
	}
}

function checkexpandimg(exp, thm) {
	if (!thm.dataset.loadingexp) {
		// no longer loading (canceled? errored out?)
		return;
	}
	// is img element ready?
	if (!exp.naturalWidth || !exp.naturalHeight) {
		// no - rethrow
		thm.dataset.loadingexp = setTimeout(checkexpandimg, 15, exp, thm);
		return;
	}

	// img element ready to show
	finishimgexpansion(exp.parentElement, exp, thm);
}

function expandimg(lnk, thm) {
	// if already loading don't mess it up
	if (thm.dataset.loadingexp)
		return;

	var exp;
	var exps = lnk.getElementsByClassName("imgexp");
	if (exps.length > 0) {
		// expanded img element already exists - reuse
		exp = exps[0];

		// note that because of previous loadingexp check
		// this should only happen with already loaded imgs
		// so don't bother with attribute polling

		// before un-hiding, remove from DOM
		lnk.removeChild(exp);
		// un-hide
		exp.style.removeProperty('display');
		// swap expanded image with thumbnail
		lnk.replaceChild(exp, thm);
		// now thm is out of DOM, hide it and put it back
		thm.style.display = 'none';
		lnk.insertBefore(thm, exp);
	} else {
		// make new expanded img element
		exp = new Image();
		exp.addEventListener('error', expandingimgerror);
		exp.src = lnk.href;
		exp.className = 'imgexp';
		exp.style.display = 'none';
		// add to DOM
		lnk.appendChild(exp);

		// mark as expanding and start polling
		thm.style.opacity = 0.75;
		thm.dataset.loadingexp = setTimeout(checkexpandimg, 15, exp, thm);
	}
}

function unexpandimg(lnk, thm, exp) {
	// before un-hiding, remove from DOM
	lnk.removeChild(thm);
	// un-hide
	thm.style.removeProperty('display');
	// swap exp -> thm
	lnk.replaceChild(thm, exp);
	// hide exp
	exp.style.display = 'none';
	// add hidden (if it didn't fail before)
	if (!exp.dataset.failed)
		lnk.appendChild(exp);

	// (attempt to) fix scroll position
	// current position from top
	var currpos = document.documentElement.scrollTop || document.body.scrollTop;
	// console.log("currpos: " + currpos);
	// current element top RELATIVE TO currpos
	var filetop = lnk.parentElement.getBoundingClientRect().top;
	// if we're beyond thumbnail image
	// NOTE: NOT whole post but just specific thumbnail
	if (filetop < 0) {
		// scroll to it. -18 to cover a bit of content above
		var newpos = currpos + filetop - 18;
		document.documentElement.scrollTop = newpos;
		document.body.scrollTop            = newpos;
	}
}

function expandaudio(lnk, thm) {
	var adiv = document.createElement('div');
	adiv.className = 'audioembed';
	adiv.style.backgroundImage = 'url("' + thm.src + '")';
	var audio = new Audio(lnk.href);
	audio.controls = true;
	adiv.appendChild(audio);

	var cspan = document.createElement('span');
	cspan.appendChild(document.createTextNode(' ['));
	clink = document.createElement('a');
		clink.href = "";
		clink.className = 'embedclose';
		clink.appendChild(document.createTextNode('Close'));
	cspan.appendChild(clink);
	cspan.appendChild(document.createTextNode(']'));

	var lpar = lnk.parentElement;
	lpar.replaceChild(adiv, lnk);
	lnk.style.display = 'none';
	lpar.insertBefore(lnk, adiv);

	// yeh this aint atomic but very likely won't be noticed
	var imginfo = lpar.getElementsByClassName("imginfo")[0];
	imginfo.appendChild(cspan);

	audio.play();
}

function dothumbclick(e, lnk, thm) {
	var typ = lnk.dataset.type;
	//console.log(">image thumb clicked, type=" + typ);
	if (typ == 'image') {
		// do expansion
		expandimg(lnk, thm);
		// don't actually open link
		e.preventDefault();
	} else if (typ == 'audio') {
		// do expansion
		expandaudio(lnk, thm);
		// don't actually open link
		e.preventDefault();
	}
}

function onglobalclick(e) {
	var tgt = e.target;
	switch (tgt.className) {
		case 'imgthumb':
			dothumbclick(e, tgt.parentElement, tgt);
			break;
		case 'imgexp':
		{
			// if we encounter this type then we already know that this is image
			var lnk = tgt.parentElement;
			var thm = lnk.getElementsByClassName('imgthumb')[0];
			unexpandimg(lnk, thm, tgt);
			// don't actually open link
			e.preventDefault();
			break;
		}
		case 'imglink':
		{
			// paranoia
			var thm = tgt.getElementsByClassName('imgthumb')[0];
			if (thm.style.display) {
				// display set (probably to none), this means we have expanded
				var exps = tgt.getElementsByClassName("imgexp");
				if (exps.length > 0) {
					unexpandimg(tgt, thm, exps[0]);
					e.preventDefault();
				}
				// else something weird happened ¯\_(ツ)_/¯
			} else {
				dothumbclick(e, tgt, thm);
			}
			break;
		}
		case 'embedclose':
		{
			// do this upfront to not forget
			e.preventDefault();
			// parent is span
			// parent of that is imginfo
			// parent of that is either opimg or rimg doesnt matter
			// it should have some sort of embed inside
			// what sort depends on hidden imglink element [which we'll need to unhide]
			var cspan = tgt.parentElement;
			var imginfo = cspan.parentElement;
			var pcont = imginfo.parentElement;
			var lnk = pcont.getElementsByClassName('imglink')[0];
			var typ = lnk.dataset.type;
			if (typ == 'audio') {
				var embs = pcont.getElementsByClassName('audioembed');
				if (embs.length > 0) {
					var emb = embs[0];
					// pause playback
					emb.childNodes[0].pause();
					// before unhiding remove from DOM
					pcont.removeChild(lnk);
					// unhide
					lnk.style.removeProperty('display');
					// replace current embed element with it
					pcont.replaceChild(lnk, emb);
					// don't reinsert emb just let it get eaten by GC

					// ohyeah also delet close button
					imginfo.removeChild(cspan);
				}
			}
		}
		case 'audioembed':
		{
			// toggle play/pause if clicked on background
			var audio = tgt.childNodes[0];
			if (audio.paused)
				audio.play();
			else
				audio.pause();
		}
	}
}

document.documentElement.addEventListener("click", onglobalclick);
