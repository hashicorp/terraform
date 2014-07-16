//
// app.js
//

var APP = (function() {

	function initialize (){
		APP.Utils.runIfClassNamePresent('page-home', initHome);
	}

	function initHome() {
		APP.Homepage.init();
	}

  	//api
	return {
		initialize: initialize
  	}

})();
