!function(){function t(h,r){var s=this instanceof t?this:e;return s.reset(r),"string"==typeof h&&h.length>0&&s.hash(h),s!==this?s:void 0}var e;t.prototype.hash=function(t){var e,h,r,s,i;switch(i=t.length,this.len+=i,h=this.k1,r=0,this.rem){case 0:h^=i>r?65535&t.charCodeAt(r++):0;case 1:h^=i>r?(65535&t.charCodeAt(r++))<<8:0;case 2:h^=i>r?(65535&t.charCodeAt(r++))<<16:0;case 3:h^=i>r?(255&t.charCodeAt(r))<<24:0,h^=i>r?(65280&t.charCodeAt(r++))>>8:0}if(this.rem=3&i+this.rem,i-=this.rem,i>0){for(e=this.h1;;){if(h=4294967295&11601*h+3432906752*(65535&h),h=h<<15|h>>>17,h=4294967295&13715*h+461832192*(65535&h),e^=h,e=e<<13|e>>>19,e=4294967295&5*e+3864292196,r>=i)break;h=65535&t.charCodeAt(r++)^(65535&t.charCodeAt(r++))<<8^(65535&t.charCodeAt(r++))<<16,s=t.charCodeAt(r++),h^=(255&s)<<24^(65280&s)>>8}switch(h=0,this.rem){case 3:h^=(65535&t.charCodeAt(r+2))<<16;case 2:h^=(65535&t.charCodeAt(r+1))<<8;case 1:h^=65535&t.charCodeAt(r)}this.h1=e}return this.k1=h,this},t.prototype.result=function(){var t,e;return t=this.k1,e=this.h1,t>0&&(t=4294967295&11601*t+3432906752*(65535&t),t=t<<15|t>>>17,t=4294967295&13715*t+461832192*(65535&t),e^=t),e^=this.len,e^=e>>>16,e=4294967295&51819*e+2246770688*(65535&e),e^=e>>>13,e=4294967295&44597*e+3266445312*(65535&e),e^=e>>>16,e>>>0},t.prototype.reset=function(t){return this.h1="number"==typeof t?t:0,this.rem=this.k1=this.len=0,this},e=new t,this.MurmurHash3=t}();

const CURSUP = "\033[A"

const [, , h="1uu8qb7", len=6] = process.argv
const HASH = parseInt(h, 36)
const LEN = +len
const CHARS = 'oenatvsilkrdpímuázjyěcbéhřýžčšůfgúňxťóďwq'.split('') // x

const radix = CHARS.length
const indexes = new Array(LEN).fill(0)
const startTime = Date.now()
const printmod = 99991 // Math.pow(radix, 4)

for (let i = 0, max = Math.pow(radix, LEN); i < max; i++) {
	const word = indexes.map(i => CHARS[i]).join('')
	if (i % printmod === 0) {
		let seconds = (Date.now() - startTime)/1000 +1
		console.log(
			CURSUP, '                                           \n',
			CURSUP,
			'>',
			`${word}\t`,
			`${(100 * i / max).toFixed(3)}%  `,
			`${eta(i / max)}  `,
			`(${((i/seconds)/1000000).toFixed(2)}M/s)  `,
			''
		)
	}	
	if (HASH === MurmurHash3(encodeURI(word).toLowerCase()).result()) {
		console.log(CURSUP,`word found: ${word}\n\n`)
		break
	}
	for (let j = 0, mod = 1; j < LEN; j++) {
		if (i % mod === mod-1) {
			indexes[j]++
			indexes[j] %= radix
		}
		mod *= radix
	}
}

function eta(progress) {
	const formatDuration = (duration) => {
		duration = duration || 1
	  duration /= 1000
	  duration -= duration % 1
	  const s = duration % 60
	  duration -= s
	  duration /= 60
	  const m = duration % 60
	  duration -= m
	  duration /= 60
	  const h = duration % 24
	  duration -= h
	  duration /= 24
	  const d = duration
	  return `${d}d ${h}h ${m}m ${s}s`
	}
	const sofar = Date.now() - startTime
	return formatDuration(sofar / progress - sofar)
}