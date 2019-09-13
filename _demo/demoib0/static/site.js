// console.log("ohayo!");

/* image expansion functionality */

function finishimgexpansion(lnk, exp, thm) {
	// mark as no longer loading
	delete thm.dataset.loadingexp;
	// remove expimg from DOM before modification
	lnk.removeChild(exp);
	// configure width and height
	if (lnk.dataset.width && lnk.dataset.height) {
		exp.width  = lnk.dataset.width;
		exp.height = lnk.dataset.height;
		/* -- eh nvm
		// if expanded img is exactly the same size as thm or v close,
		// make it have border to show difference
		if (Math.abs(exp.width - thm.width + 1) < 5 &&
			Math.abs(exp.height - thm.height + 1) < 5) {

			exp.style.border = '2px black dashed';
		}
		*/
	}
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

function newembedcontrol(me) {
	var cspan = document.createElement('span');

	// XXX it doesn't seem like media elements have event for loop var change
	// so we can't do reliable back notification

	// checkbox
	var loopcb = document.createElement('input');
	loopcb.type = 'checkbox';
	loopcb.addEventListener('input', function(e){
		me.loop = e.target.checked;
	});
	loopcb.style.margin = '0px 2px 0px 2px';
	loopcb.style.verticalAlign = 'bottom';

	// label which carries checkbox and text
	var cloop = document.createElement('label');
	cloop.appendChild(loopcb);
	cloop.appendChild(document.createTextNode('Loop'));

	// box for [] and above label element
	var cloopbox = document.createElement('span');
	cloopbox.style.display = 'inline-block';
	cloopbox.appendChild(document.createTextNode('['));
	cloopbox.appendChild(cloop);
	cloopbox.appendChild(document.createTextNode(']'));

	// put it in
	cspan.appendChild(cloopbox);

	// space before next box
	cspan.appendChild(document.createTextNode(' '));

	// close link
	var clink = document.createElement('a');
	clink.href = "";
	clink.className = 'embedclose';
	clink.appendChild(document.createTextNode('Close'));

	// close link box
	var clinkbox = document.createElement('span');
	clinkbox.style.display = 'inline-block';
	clinkbox.appendChild(document.createTextNode('['));
	clinkbox.appendChild(clink);
	clinkbox.appendChild(document.createTextNode(']'));

	// put it in
	cspan.appendChild(clinkbox);

	return cspan;
}

function expandaudio(lnk, thm) {
	var adiv = document.createElement('div');
	adiv.className = 'audioembed';
	adiv.style.backgroundImage = 'url("' + thm.src + '")';
	var audio = new Audio(lnk.href);
	audio.controls = true;
	adiv.appendChild(audio);
	var idiv = document.createElement('div');
	idiv.className = 'audioembedinner';
	adiv.appendChild(idiv);

	var cspan = newembedcontrol(audio);

	var lpar = lnk.parentElement;
	lpar.replaceChild(adiv, lnk);
	lnk.style.display = 'none';
	lpar.insertBefore(lnk, adiv);

	// yeh this aint atomic but very likely won't be noticed
	var imginfo = lpar.getElementsByClassName("imginfo")[0];
	imginfo.appendChild(cspan);

	audio.play();
}

function expandvideo(lnk, thm) {
	var video = document.createElement('video');
	video.src = lnk.href;
	video.className = 'videoembed';
	video.controls = true;

	var cspan = newembedcontrol(video);

	// TODO maybe wait for loadedmetadata event before showing
	var lpar = lnk.parentElement;
	lpar.replaceChild(video, lnk);
	lnk.style.display = 'none';
	lpar.insertBefore(lnk, video);

	// yeh this aint atomic but very likely won't be noticed
	var imginfo = lpar.getElementsByClassName("imginfo")[0];
	imginfo.appendChild(cspan);

	video.play();
}


function dothumbclick(e, lnk, thm) {
	var typ = lnk.dataset.type;
	//console.log(">image thumb clicked, type=" + typ);
	switch (typ) {
		case 'image':
			// do expansion
			expandimg(lnk, thm);
			break;
		case 'audio':
			// do expansion
			expandaudio(lnk, thm);
			break;
		case 'video':
			// do expansion
			expandvideo(lnk, thm);
			break;
		default:
			return;
	}
	// don't actually open link
	e.preventDefault();
}

function refer(refcont) {
	//console.log("ref click happen, post: " + refcont);
	var ta = document.getElementById("message");
	if (ta) {
		//console.log("found message element");

		var currtxt = ta.value;
		var selend = ta.selectionEnd;
		var endspace = false;
		var txtinsert = "";
		if (selend > 0) {
			// peek previous character
			var pch = currtxt[selend-1];
			if (pch != '\n') {
				endspace = true;
				if (pch != ' ' && pch != '\t') {
					txtinsert += " ";
				}
			}
		}
		txtinsert += ">>" + refcont;
		if (!endspace) {
			txtinsert += '\n';
			// also quote selected text content
			var sel = window.getSelection();
			var str = sel.toString();
			var stra = str.split(/\n/);
			stra = stra.map(function(s){
				// trim line endings
				while (s.length > 0) {
					var l = s.length - 1;
					var c = s[l];
					if (c != '\n' && c != '\r' && c != ' ' && c != '\t')
						break;
					s = s.substring(0, l);
				}
				// only if line is non-empty, quote it
				if (s.length > 0)
					return '>' + s;
				else
					return s;
			});
			// trim trailing empty lines
			for (var i = stra.length-1; i >= 0; i--) {
				if (stra[i].length > 0)
					break;

				stra.length--;
			}
			// only append if we have something to append
			if (stra.length > 0)
				txtinsert += stra.join('\n') + '\n';
		}
		else
			txtinsert += " ";

		ta.setRangeText(txtinsert, selend, selend, "end");
		ta.focus();
	}
	return false;
}


var doupdate = false;

/*
function findAllChildrenByClass(el, name, act) {
	for (var i = 0; i < el.childNodes.length; i++) {
		var e = el.childNodes[i];
		if (e.className == name)
			act(e);
		else
			findAllChildrenByClass(e, name, act);
	}
}
*/

function updateBackRefs(exinfo, elinfo) {
	var exbrefs = exinfo.getElementsByClassName("bref");
	var elbrefs = elinfo.getElementsByClassName("bref");
	var i = 0;
	console.log("brefs update: start");
	// first check existing
	for (; i < exbrefs.length; i++) {
		var exbref = exbrefs[i];

		// check for difference
		if (i >= elbrefs.length ||
			elbrefs[i].textContent != exbref.textContent) {

			// it differs. clear this and all following
			console.log("deleting brefs since " + exbref.textContent);
			// removeChild modifies exbrefs object... which is pretty crazy if you ask me
			// just work it around using reverse loop, as I don't wanna rely on that
			for (var j = exbrefs.length-1; j >= i; j--) {
				console.log("delet bref " + exbrefs[j].textContent);
				// delet whitespace text node
				var ps = exbrefs[j].previousSibling;
				if (ps && ps.nodeType == 3) {
					exinfo.removeChild(ps);
				}
				// delet element
				exinfo.removeChild(exbrefs[j]);
			}
			break;
		}
	}
	// add any extra/different
	for (; i < elbrefs.length; i++) {
		console.log("adding new bref " + elbrefs[i].textContent);
		exinfo.appendChild(document.createTextNode(' '));
		exinfo.appendChild(elbrefs[i]);
	}
	console.log("brefs update: end");
}

function processNewReply(el) {
	console.log("processNewReply: id=" + el.id);
	var ex = document.getElementById(el.id);
	if (ex) {
		console.log("processNewReply: got ex");
		// update backrefs of reply
		updateBackRefs(
			ex.getElementsByClassName("rinfo")[0],
			el.getElementsByClassName("rinfo")[0]);
	}
	else {
		console.log("processNewReply: got no ex");
		// add whole new element
		// we actually gotta add reply-outer element
		var bottom = document.getElementById("bottom");
		bottom.parentElement.insertBefore(el.parentElement, bottom);
	}
}

function processUpdatedThread(nel) {
	console.log("processUpdatedThread: start");

	// update backrefs of OP
	updateBackRefs(
		document.getElementsByClassName("opinfo")[0],
		nel.getElementsByClassName("opinfo")[0]);

	var repls = nel.getElementsByClassName("reply");
	for (var i = 0; i < repls.length; i++) {
		processNewReply(repls[i]);
	}

	console.log("processUpdatedThread: end");
}

function updateclick(e, tgt) {
	// to prevent usage before feature is finished
	if (!doupdate)
		return;

	e.preventDefault();

	console.log("update clicked");

	var loc = window.location;
	var thispageurl = loc.pathname + loc.search;
	console.log("url: " + thispageurl);

	var req = new XMLHttpRequest();
	req.addEventListener("load", function() {
		console.log("req loaded!");

		processUpdatedThread(req.responseXML);
	});
	req.addEventListener("error", function() {
		console.log("req erred!");
	});
	req.open("GET", thispageurl);
	req.responseType = "document";
	req.send();
	console.log("update request sent");
}

function togglePause(audio) {
	if (audio.paused)
		audio.play();
	else
		audio.pause();
}

function onglobalclick(e) {
	var tgt = e.target;
	switch (tgt.className) {
		case 'update':
			updateclick(e, tgt);
			break;
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
		case 'pnum_ref':
		{
			refer(tgt.textContent);
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
			var cspan = tgt.parentElement.parentElement;
			var imginfo = cspan.parentElement;
			var pcont = imginfo.parentElement;
			var lnk = pcont.getElementsByClassName('imglink')[0];
			var typ = lnk.dataset.type;
			var emb;
			if (typ == 'audio') {
				var embs = pcont.getElementsByClassName('audioembed');
				if (embs.length > 0) {
					emb = embs[0];
					// pause playback
					emb.childNodes[0].pause();
				}
			} else if (typ == 'video') {
				var embs = pcont.getElementsByClassName('videoembed');
				if (embs.length > 0) {
					emb = embs[0];
					// pause playback
					emb.pause();
				}
			}
			if (emb) {
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
			break;
		}
		case 'audioembed':
		{
			// toggle play/pause if clicked on background
			togglePause(tgt.childNodes[0]);
			e.preventDefault();
			break;
		}
		case 'audioembedinner':
		{
			// toggle play/pause if clicked on background
			togglePause(tgt.parentElement.childNodes[0]);
			e.preventDefault();
			break;
		}
	}
}

document.documentElement.addEventListener("click", onglobalclick);

// executed once page is ready
function ready() {
	// linkify all of pnum_ref elements
	var els = document.getElementsByClassName('pnum_ref');
	for (var i = 0; i < els.length; i++) {
		var e = els[i];
		if (e.tagName.toLowerCase() == "a") {
			e.href = '#';
		}
	}
}

// executed once page completed loading
function completed() {
	document.removeEventListener("DOMContentLoaded", completed);
	window.removeEventListener("load", completed);
	ready();
}

// stolen from jquery
if (document.readyState === "complete" ||
	(document.readyState !== "loading" &&
		!document.documentElement.doScroll)) {

	// just call it now
	// don't delay as we don't do much stuff in there
	ready();

} else {
	// Use the handy event callback
	document.addEventListener("DOMContentLoaded", completed);
	// A fallback to window.onload, that will always work
	window.addEventListener("load", completed);
}
