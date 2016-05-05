var http = require('http')

exports.handler = function(event, context) {
    http.get("http://requestb.in/MODIFIED", function(res) {
        console.log("success", res.statusCode, res.body)
    }).on('error', function(e) {
        console.log("error", e)
    })
}
