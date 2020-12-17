/*
 * FreeSec: libcrypt for NetBSD
 *
 * Copyright (c) 1994 David Burren
 * All rights reserved.
 *
 * Adapted for FreeBSD-2.0 by Geoffrey M. Rehmet
 *      this file should now *only* export crypt(), in order to make
 *      binaries of libcrypt exportable from the USA
 *
 * Adapted for FreeBSD-4.0 by Mark R V Murray
 *      this file should now *only* export crypt_des(), in order to make
 *      a module that can be optionally included in libcrypt.
 *
 * Adopted for nksrv by cathugger
 *      this file is now rewritten in golang.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions
 * are met:
 * 1. Redistributions of source code must retain the above copyright
 *    notice, this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright
 *    notice, this list of conditions and the following disclaimer in the
 *    documentation and/or other materials provided with the distribution.
 * 3. Neither the name of the author nor the names of other contributors
 *    may be used to endorse or promote products derived from this software
 *    without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE AUTHOR AND CONTRIBUTORS ``AS IS'' AND
 * ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED.  IN NO EVENT SHALL THE AUTHOR OR CONTRIBUTORS BE LIABLE
 * FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
 * DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS
 * OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
 * HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
 * LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY
 * OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF
 * SUCH DAMAGE.
 *
 * This is an original implementation of the DES and the crypt(3) interfaces
 * by David Burren <davidb@werj.com.au>.
 *
 * An excellent reference on the underlying algorithm (and related
 * algorithms) is:
 *
 *      B. Schneier, Applied Cryptography: protocols, algorithms,
 *      and source code in C, John Wiley & Sons, 1994.
 *
 * Note that in that book's description of DES the lookups for the initial,
 * pbox, and final permutations are inverted (this has been brought to the
 * attention of the author).  A list of errata for this book has been
 * posted to the sci.crypt newsgroup by the author and is available for FTP.
 */

package unixdescrypt

import (
	"encoding/binary"
	"sync"
)

var init_perm, final_perm [64]byte
var m_sbox [4][4096]byte
var psbox [4][256]uint32
var ip_maskl, ip_maskr [8][256]uint32
var fp_maskl, fp_maskr [8][256]uint32
var key_perm_maskl, key_perm_maskr [8][128]uint32
var comp_maskl, comp_maskr [8][128]uint32

var des_init_once sync.Once

func ascii_to_bin(ch byte) byte {
	if ch > 'z' {
		return 0
	}
	if ch >= 'a' {
		return ch - 'a' + 38
	}
	if ch > 'Z' {
		return 0
	}
	if ch >= 'A' {
		return ch - 'A' + 12
	}
	if ch > '9' {
		return 0
	}
	if ch >= '.' {
		return ch - '.'
	}
	return 0
}

type des_context struct {
	en_keysl, en_keysr [16]uint32
	saltbits           uint32
}

func des_init() {
	var i, j, b, k int
	var inbit int
	var obit byte
	var p, il, ir, fl, fr *uint32

	/*
	 * Invert the S-boxes, reordering the input bits.
	 */
	for i = 0; i < 8; i++ {
		for j = 0; j < 64; j++ {
			b = (j & 0x20) | ((j & 1) << 4) | ((j >> 1) & 0xf)
			u_sbox[i][j] = sbox[i][b]
		}
	}

	/*
	 * Convert the inverted S-boxes into 4 arrays of 8 bits.
	 * Each will handle 12 bits of the S-box input.
	 */
	for b = 0; b < 4; b++ {
		for i = 0; i < 64; i++ {
			for j = 0; j < 64; j++ {
				m_sbox[b][(i<<6)|j] =
					(u_sbox[(b << 1)][i] << 4) | u_sbox[(b<<1)+1][j]
			}
		}
	}

	/*
	 * Set up the initial & final permutations into a useful form, and
	 * initialise the inverted key permutation.
	 */
	for i = 0; i < 64; i++ {
		x := ip[i] - 1
		final_perm[i] = x
		init_perm[x] = byte(i)
		inv_key_perm[i] = 255
	}

	/*
	 * Invert the key permutation and initialise the inverted key
	 * compression permutation.
	 */
	for i = 0; i < 56; i++ {
		inv_key_perm[key_perm[i]-1] = byte(i)
		inv_comp_perm[i] = 255
	}

	/*
	 * Invert the key compression permutation.
	 */
	for i = 0; i < 48; i++ {
		inv_comp_perm[comp_perm[i]-1] = byte(i)
	}

	/*
	 * Set up the OR-mask arrays for the initial and final permutations,
	 * and for the key initial and compression permutations.
	 */
	for k = 0; k < 8; k++ {
		for i = 0; i < 256; i++ {
			il = &ip_maskl[k][i]
			*il = 0
			ir = &ip_maskr[k][i]
			*ir = 0
			fl = &fp_maskl[k][i]
			*fl = 0
			fr = &fp_maskr[k][i]
			*fr = 0
			for j = 0; j < 8; j++ {
				inbit = 8*k + j
				if (byte(i) & bits8[j]) != 0 {
					if obit = init_perm[inbit]; obit < 32 {
						*il |= bits32[obit]
					} else {
						*ir |= bits32[obit-32]
					}
					if obit = final_perm[inbit]; obit < 32 {
						*fl |= bits32[obit]
					} else {
						*fr |= bits32[obit-32]
					}
				}
			}
		}
		for i = 0; i < 128; i++ {
			il = &key_perm_maskl[k][i]
			*il = 0
			ir = &key_perm_maskr[k][i]
			*ir = 0
			for j = 0; j < 7; j++ {
				inbit = 8*k + j
				if (byte(i) & bits8[j+1]) != 0 {
					obit = inv_key_perm[inbit]
					if obit == 255 {
						continue
					}
					if obit < 28 {
						*il |= bits28[obit]
					} else {
						*ir |= bits28[obit-28]
					}
				}
			}
			il = &comp_maskl[k][i]
			*il = 0
			ir = &comp_maskr[k][i]
			*ir = 0
			for j = 0; j < 7; j++ {
				inbit = 7*k + j
				if (byte(i) & bits8[j+1]) != 0 {
					obit = inv_comp_perm[inbit]
					if obit == 255 {
						continue
					}
					if obit < 24 {
						*il |= bits24[obit]
					} else {
						*ir |= bits24[obit-24]
					}
				}
			}
		}
	}

	/*
	 * Invert the P-box permutation, and convert into OR-masks for
	 * handling the output of the S-box arrays setup above.
	 */
	for i = 0; i < 32; i++ {
		un_pbox[pbox[i]-1] = byte(i)
	}

	for b = 0; b < 4; b++ {
		for i = 0; i < 256; i++ {
			p = &psbox[b][i]
			*p = 0
			for j = 0; j < 8; j++ {
				if (byte(i) & bits8[j]) != 0 {
					*p |= bits32[un_pbox[8*b+j]]
				}
			}
		}
	}
}

func (ctx *des_context) setup_salt(salt uint32) {

	var obit, saltbit uint32

	ctx.saltbits = 0
	saltbit = 1
	obit = 0x800000
	for i := 0; i < 24; i++ {
		if (salt & saltbit) != 0 {
			ctx.saltbits |= obit
		}
		saltbit <<= 1
		obit >>= 1
	}
}

func (ctx *des_context) des_setkey(key [8]byte) {

	rawkey0 := binary.BigEndian.Uint32(key[:4])
	rawkey1 := binary.BigEndian.Uint32(key[4:])

	/*
	 * Do key permutation and split into two 28-bit subkeys.
	 */
	k0 := 0 |
		key_perm_maskl[0][rawkey0>>25] |
		key_perm_maskl[1][(rawkey0>>17)&0x7f] |
		key_perm_maskl[2][(rawkey0>>9)&0x7f] |
		key_perm_maskl[3][(rawkey0>>1)&0x7f] |
		key_perm_maskl[4][rawkey1>>25] |
		key_perm_maskl[5][(rawkey1>>17)&0x7f] |
		key_perm_maskl[6][(rawkey1>>9)&0x7f] |
		key_perm_maskl[7][(rawkey1>>1)&0x7f]

	k1 := 0 |
		key_perm_maskr[0][rawkey0>>25] |
		key_perm_maskr[1][(rawkey0>>17)&0x7f] |
		key_perm_maskr[2][(rawkey0>>9)&0x7f] |
		key_perm_maskr[3][(rawkey0>>1)&0x7f] |
		key_perm_maskr[4][rawkey1>>25] |
		key_perm_maskr[5][(rawkey1>>17)&0x7f] |
		key_perm_maskr[6][(rawkey1>>9)&0x7f] |
		key_perm_maskr[7][(rawkey1>>1)&0x7f]

	/*
	 * Rotate subkeys and do compression permutation.
	 */
	shifts := byte(0)
	for round := 0; round < 16; round++ {

		shifts += key_shifts[round]

		t0 := (k0 << shifts) | (k0 >> (28 - shifts))
		t1 := (k1 << shifts) | (k1 >> (28 - shifts))

		ctx.en_keysl[round] = 0 |
			comp_maskl[0][(t0>>21)&0x7f] |
			comp_maskl[1][(t0>>14)&0x7f] |
			comp_maskl[2][(t0>>7)&0x7f] |
			comp_maskl[3][t0&0x7f] |
			comp_maskl[4][(t1>>21)&0x7f] |
			comp_maskl[5][(t1>>14)&0x7f] |
			comp_maskl[6][(t1>>7)&0x7f] |
			comp_maskl[7][t1&0x7f]

		ctx.en_keysr[round] = 0 |
			comp_maskr[0][(t0>>21)&0x7f] |
			comp_maskr[1][(t0>>14)&0x7f] |
			comp_maskr[2][(t0>>7)&0x7f] |
			comp_maskr[3][t0&0x7f] |
			comp_maskr[4][(t1>>21)&0x7f] |
			comp_maskr[5][(t1>>14)&0x7f] |
			comp_maskr[6][(t1>>7)&0x7f] |
			comp_maskr[7][t1&0x7f]
	}
}

func (ctx *des_context) do_des(l_in, r_in uint32, count int) (l_out, r_out uint32) {
	/*
	 * l_in, r_in, l_out, and r_out are in pseudo-"big-endian" format.
	 */

	var l, r uint32
	var f, r48l, r48r uint32

	/*
	 * Do initial permutation (IP).
	 */
	l = 0 |
		ip_maskl[0][l_in>>24] |
		ip_maskl[1][(l_in>>16)&0xff] |
		ip_maskl[2][(l_in>>8)&0xff] |
		ip_maskl[3][l_in&0xff] |
		ip_maskl[4][r_in>>24] |
		ip_maskl[5][(r_in>>16)&0xff] |
		ip_maskl[6][(r_in>>8)&0xff] |
		ip_maskl[7][r_in&0xff]
	r = 0 |
		ip_maskr[0][l_in>>24] |
		ip_maskr[1][(l_in>>16)&0xff] |
		ip_maskr[2][(l_in>>8)&0xff] |
		ip_maskr[3][l_in&0xff] |
		ip_maskr[4][r_in>>24] |
		ip_maskr[5][(r_in>>16)&0xff] |
		ip_maskr[6][(r_in>>8)&0xff] |
		ip_maskr[7][r_in&0xff]

	for ii := 0; ii < count; ii++ {
		/*
		 * Do each round.
		 */
		for i := 0; i < 16; i++ {
			/*
			 * Expand R to 48 bits (simulate the E-box).
			 */
			r48l = 0 |
				((r & 0x00000001) << 23) |
				((r & 0xf8000000) >> 9) |
				((r & 0x1f800000) >> 11) |
				((r & 0x01f80000) >> 13) |
				((r & 0x001f8000) >> 15)

			r48r = 0 |
				((r & 0x0001f800) << 7) |
				((r & 0x00001f80) << 5) |
				((r & 0x000001f8) << 3) |
				((r & 0x0000001f) << 1) |
				((r & 0x80000000) >> 31)

			/*
			 * Do salting for crypt() and friends, and
			 * XOR with the permuted key.
			 */
			f = (r48l ^ r48r) & ctx.saltbits
			r48l ^= f ^ ctx.en_keysl[i]
			r48r ^= f ^ ctx.en_keysr[i]

			/*
			 * Do sbox lookups (which shrink it back to 32 bits)
			 * and do the pbox permutation at the same time.
			 */
			f = 0 |
				psbox[0][m_sbox[0][r48l>>12]] |
				psbox[1][m_sbox[1][r48l&0xfff]] |
				psbox[2][m_sbox[2][r48r>>12]] |
				psbox[3][m_sbox[3][r48r&0xfff]]

			/*
			 * Now that we've permuted things, complete f().
			 */
			f ^= l
			l = r
			r = f
		}

		r = l
		l = f
	}

	/*
	 * Do final permutation (inverse of IP).
	 */
	l_out = 0 |
		fp_maskl[0][l>>24] |
		fp_maskl[1][(l>>16)&0xff] |
		fp_maskl[2][(l>>8)&0xff] |
		fp_maskl[3][l&0xff] |
		fp_maskl[4][r>>24] |
		fp_maskl[5][(r>>16)&0xff] |
		fp_maskl[6][(r>>8)&0xff] |
		fp_maskl[7][r&0xff]
	r_out = 0 |
		fp_maskr[0][l>>24] |
		fp_maskr[1][(l>>16)&0xff] |
		fp_maskr[2][(l>>8)&0xff] |
		fp_maskr[3][l&0xff] |
		fp_maskr[4][r>>24] |
		fp_maskr[5][(r>>16)&0xff] |
		fp_maskr[6][(r>>8)&0xff] |
		fp_maskr[7][r&0xff]
	return
}

func CryptDES(key []byte, saltstr [2]byte, outbuf []byte) []byte {
	var salt, l uint32
	var keybuf [8]byte

	des_init_once.Do(des_init)

	/*
	 * Copy the key, shifting each character up by one bit
	 * and padding with zeros.
	 */
	for i := 0; i < 8; i++ {
		if i >= len(key) {
			// rest are zeros anyway
			break
		}
		keybuf[i] = key[i] << 1
	}

	var ctx des_context

	ctx.des_setkey(keybuf)

	/*
	 * "old"-style:
	 *      setting - 2 bytes of salt
	 *      key - up to 8 characters
	 */

	salt = (uint32(ascii_to_bin(saltstr[1])) << 6) |
		uint32(ascii_to_bin(saltstr[0]))
	outbuf = append(outbuf, saltstr[0])
	outbuf = append(outbuf, saltstr[1])

	ctx.setup_salt(salt)

	/*
	 * Do it.
	 */
	r0, r1 := ctx.do_des(0, 0, 25)

	/*
	 * Now encode the result...
	 */
	l = r0 >> 8
	outbuf = append(outbuf, ascii64[(l>>18)&0x3f])
	outbuf = append(outbuf, ascii64[(l>>12)&0x3f])
	outbuf = append(outbuf, ascii64[(l>>6)&0x3f])
	outbuf = append(outbuf, ascii64[l&0x3f])

	l = (r0 << 16) | ((r1 >> 16) & 0xffff)
	outbuf = append(outbuf, ascii64[(l>>18)&0x3f])
	outbuf = append(outbuf, ascii64[(l>>12)&0x3f])
	outbuf = append(outbuf, ascii64[(l>>6)&0x3f])
	outbuf = append(outbuf, ascii64[l&0x3f])

	l = r1 << 2
	outbuf = append(outbuf, ascii64[(l>>12)&0x3f])
	outbuf = append(outbuf, ascii64[(l>>6)&0x3f])
	outbuf = append(outbuf, ascii64[l&0x3f])

	return outbuf
}
