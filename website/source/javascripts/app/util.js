//
// util.js
//
var APP = APP || {};

APP.Utils =  (function () {
	return {
	//check for mobile user agents
	  	isMobile : (function(){
	           if( navigator.userAgent.match(/Android/i)
	           || navigator.userAgent.match(/webOS/i)
	           || navigator.userAgent.match(/iPhone/i)
	           //|| navigator.userAgent.match(/iPad/i)
	           || navigator.userAgent.match(/iPod/i)
	           || navigator.userAgent.match(/BlackBerry/i)
	           || navigator.userAgent.match(/Windows Phone/i)
	           ){
	                  return true;
	            }
	           else {
	              return false;
	            }
	  	})(),

		runIfClassNamePresent: function(selector, initFunction) {
	        var elms = document.getElementsByClassName(selector);
	        if (elms.length > 0) {
	            initFunction();
	        }
	    }
	}

}());