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
;//
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

}());;//homepage.js

var APP = APP || {};

(function () {
  APP.Homepage = (function () {
    return {

      ui : null,

      init: function () {
        var _this = this;

        //cache elements
        this.ui = {
          $doc: $(window),
          $hero: $('#jumbotron'),
          $collapse: $('.navbar-collapse')
        }
        
        this.addEventListeners();

      },

      addEventListeners: function(){
        var _this = this;

        if(APP.Utils.isMobile)
          return;
        
        _this.ui.$doc.scroll(function() {

          //if collapseable menu is open dont do parrallax. It looks wonky. Bootstrap conflict
          if( _this.ui.$collapse.hasClass('in'))
              return;

          var top = _this.ui.$doc.scrollTop(),
              speedAdj = (top*0.8),
              speedAdjOffset = speedAdj - top;

          _this.ui.$hero.css('webkitTransform', 'translate(0, '+ speedAdj +'px)');
          _this.ui.$hero.find('.container').css('webkitTransform', 'translate(0, '+  speedAdjOffset +'px)');
        })
      }
    }
  }());

}(jQuery, this));

