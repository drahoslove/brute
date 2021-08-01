extractToBrute = (x) => eval((
  document.querySelector(".tajenka-obal script") ||
  (x=1, document.querySelectorAll("body>script")[1])
).innerText.substr(13))
  .hashe
  .map(str => str.split(','))
  .map((hashes, i, arr) => ({
    len: i === 0 ? arr.length-1 : hashes.length-(x?1:2),
    hash: hashes[0]
  }))
  .map(({hash, len}) => `./brute ${hash} ${len}`)
  .join('\n')
