(function(){

var Init = {

  start: function(){ 
    var classname = this.hasClass(document.body, 'page-sub');

    if (classname) {
      this.addEventListeners();
    }
  },

  hasClass: function (elem, className) {
    return new RegExp(' ' + className + ' ').test(' ' + elem.className + ' ');
  },

  addEventListeners: function(){
    var _this = this;
    //console.log(document.querySelectorAll('.navbar-static-top')[0]);
    window.addEventListener('resize', _this.resizeImage, false);

    this.resizeImage();
  },

  resizeImage: function(){

    var header = document.getElementById('header'),
        footer = document.getElementById('footer-wrap'),
        main = document.getElementById('main-content'),
        vp = window.innerHeight,
        bodyHeight = document.body.clientHeight,
        hHeight = header.clientHeight,
        fHeight = footer.clientHeight,
        withMinHeight = hHeight + fHeight + 830;

    if(withMinHeight >  bodyHeight ){
      var newHeight = (vp - (hHeight+fHeight)) + 'px';
      main.style.height = newHeight;
    }    
  }

};

Init.start();

})();

(function($){
  // functions to handle the documentation scroll bar
  // these functions will ensure that
  //   the selected link is scrolled to on page load (ie visible so the user does not loose context)
  //   the documentation side bar does not overlap the footer or header
  //
  // these functions should only be applied to a 'desktop' display – the mobile layout is incompatible

  // scrolls the nav to the active link in the side bar
  // called only on load
  function scrollNavIntoContext(sb){
    var path = window.location.pathname
      , link = sb.find("a[href='"+path+"']")
      , padding = link.height() * 2;

    if(link[0]){
      sb.scrollTop(
        (link.offset().top - (sb.offset().top + padding))
      )
    }
  }

  // this function will resize the top and the bottom of the side bar to fit nicely
  // between the header and footer.
  //
  // it would be great to avoid this level of complexity, but I do not know of a CSS rule
  // set to pull this off – if there is one.
  function monitorScroll(sb){
    var header = $('#header')
      , footer = $('#footer')
      , doc = $(document)
      , win = $(window)
      , p   // current scroll position
      , hb  // header bottom
      , ft; // footer top

    $(window).on('resize', function(){
      hb = header.offset().top + header.height();
      ft = footer.offset().top;
    })

    $(window, document).on('scroll resize',function(){
      // do nothing for mobile/small screens
      // the css will define the position via media rules
      if(sb.css('position') != 'fixed') { return; }

      p = doc.scrollTop();
      sb.css('top', Math.max(hb - p, 0)+'px')
      sb.css('bottom', Math.max((p - (ft - win.height())), 0)+'px')
    })
  }

  function monitorResize(sb){
    $(window, document).on('scroll', function(){
      sb.css('width', sb.parent().width())
    })
  }

  function init(){
    var sb = $('.docs-sidebar');

    scrollNavIntoContext(sb);
    monitorScroll(sb);
    monitorResize(sb);
    $(window).scroll().resize();
  }

  $(init);
})(jQuery);
