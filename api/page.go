package api

import (
	"bytes"
	"strconv"
)

var (
	home1 = []byte(`
<!DOCTYPE html>
<html>
<head>
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Strings on Ethereum</title>
<style>
body {margin:0;}

.navbar {
  overflow: hidden;
  background-color: #444;
  position: fixed;
  top: 0;
  width: 100%;
}

.navbar form {
  float: left;
  display: block;
  color: #f2f2f2;
  text-align: center;
  padding: 14px 16px;
  text-decoration: none;
  font-size: 17px;
}

.navbar a {
  float: left;
  display: block;
  color: #f2f2f2;
  text-align: center;
  padding: 14px 16px;
  text-decoration: none;
  font-size: 17px;
}

.navbar a:hover {
  background: #ddd;
  color: black;
}

.main {
  padding: 16px;
  margin-top: 30px;
  height: 100%;
  overflow: auto;
}
</style>
</head>
<body>
<div class="navbar">
  <a href="./">Home[Beta testing]</a>
  <a href="https://github.com/">Source Code (soon to upload)</a>
  <form action="start" method="GET">
    <input type="text" name="blocknum" id="blknuminput">
    <input type="submit" value="strings in block">
  </form>
</div>

<div class="main">
<table border='1' id='text_table'></table>
</div>

<script>
var lastBlockNum;
lastBlockNum =  `)

	home2 = []byte(`;
window.addEventListener('wheel', function(ev) {
    if ((window.innerHeight + window.pageYOffset) >= document.body.offsetHeight) {
        get(lastBlockNum+1)
    }
});

function get(startnum){

  var xmlhttp, recs, txt = "";
  xmlhttp = new XMLHttpRequest();
  xmlhttp.onreadystatechange = function() {
    if (this.readyState == 4 && this.status == 200) {
      recs = JSON.parse(this.responseText);
      var tbl = document.getElementById('text_table')
      for (var i =0; i < recs.length; i++) {
        for (var j=0; j < recs[i].Text.length;j++) {
          var row = tbl.insertRow(tbl.rows.length)
          var cell1 = row.insertCell(0)
          cell1.innerHTML = "<a href='https://etherscan.io/block/" + recs[i].BlockNum + "'>Block " + recs[i].BlockNum + "</a>"
          var cell2 = row.insertCell(1)
          cell2.innerHTML = "<a href='https://etherscan.io/tx/" + recs[i].Text[j].Txn + "'>txn " + recs[i].Text[j].Txn.slice(0, 4) + "..." + recs[i].Text[j].Txn.slice(-4) + "</a>"
          var cell3 = row.insertCell(2)
          cell3.innerHTML = recs[i].Text[j].Text
          lastBlockNum = recs[i].BlockNum
        }
      }
      if (recs.length === 0) {
          lastBlockNum = lastBlockNum + 1
      }
    }
  };

  xmlhttp.open("GET", "text?blocknum="+startnum, true);
  xmlhttp.send(null)
};

get(lastBlockNum)

</script>
</body>
</html>
`)
)

func getPage(startNum uint64) []byte {
	num := []byte(strconv.FormatUint(startNum, 10))
	var buf bytes.Buffer
	buf.Write(home1)
	buf.Write(num)
	buf.Write(home2)
	return buf.Bytes()
}
