package textcaptcha

type Answer string
type Distortion []byte

/*
 * layout for key:
 * 8[keyid]|24[nonce]|?[encrypted data]
 * leyout for data:
 * 2[type]|8[expiredate]|1[x]|x[keydata]|?[random]
 */
