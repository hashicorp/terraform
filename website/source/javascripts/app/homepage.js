//homepage.js

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

