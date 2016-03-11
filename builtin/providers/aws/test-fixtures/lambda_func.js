var http = require('http')

exports.handler = function(event, context) {
    http.get("http://requestb.in/10m32wg1", function(res) {
        console.log("success", res.statusCode, res.body)
    }).on('error', function(e) {
        console.log("error", e)
    })
}
