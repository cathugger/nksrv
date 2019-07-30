// console.log("ohayo!");

function onglobalclick(e) {
	e = e || window.event;
	var tgt = e.target;
	if (tgt.className == "imgthumb") {
		var lnk = tgt.parentElement;
		var typ = lnk.dataset.type;
		//console.log(">image thumb clicked, type=" + typ);
		if (typ == 'image') {
			// atomic replace will need full copy as we gonna modify child elements
			var newlnk = lnk.cloneNode(true);
			newlnk.getElementsByClassName('imgthumb')[0].style.display = 'none';
			var exps = newlnk.getElementsByClassName("imgexp");
			if (exps.length > 0) {
				// expansions already exist
				exps[0].style.removeProperty('display');
			} else {
				// new expansion
				var exp = document.createElement('img');
				exp.src = lnk.href;
				// img ones are to hint browser real size
				// style ones are because max-width otherwise axes ratio
				exp.width  = lnk.dataset.width;
				exp.height = lnk.dataset.height;
				exp.style.width = exp.style.height = "auto";
				exp.className = 'imgexp';
				newlnk.appendChild(exp);
			}
			var lp = lnk.parentElement;
			lp.replaceChild(newlnk, lnk);

			e.preventDefault();
		}
	} else if (tgt.className == "imgexp") {
		// if we encounter this type then we already know that this is image
		var lnk = tgt.parentElement;
		var newlnk = lnk.cloneNode(true);
		newlnk.getElementsByClassName('imgthumb')[0].style.removeProperty('display');
		newlnk.getElementsByClassName('imgexp')[0].style.display = 'none';
		var lp = lnk.parentElement;
		lp.replaceChild(newlnk, lnk);

		// current position from top
		var currpos = document.documentElement.scrollTop || document.body.scrollTop;
		// console.log("currpos: " + currpos);
		// current element top RELATIVE TO currpos, minus some additional stuff to feel more natural
		var filetop = newlnk.parentElement.getBoundingClientRect().top - 18;
		
		// if we're beyond thumbnail image
		// NOTE: NOT whole post but just specific thumbnail
		if (filetop < 0) {
			// scroll to it
			var newpos = currpos + filetop;
			document.documentElement.scrollTop = newpos;
			document.body.scrollTop            = newpos;
		}

		e.preventDefault();
	}
}

document.addEventListener("click", onglobalclick);
