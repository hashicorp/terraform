(function(){

var Init = {

  start: function(){
    var classname = this.hasClass(document.body, 'page-sub');

    if (classname) {
      this.addEventListeners();
    }

    this.makeNavSticky();
  },

  hasClass: function (elem, className) {
    return new RegExp(' ' + className + ' ').test(' ' + elem.className + ' ');
  },

  addEventListeners: function(){
    var _this = this;
    //console.log(document.querySelectorAll('.navbar-static-top')[0]);
    window.addEventListener('resize', _this.onResize, false);

    this.resizeImage();
  },

  onResize: function() {
    this.resizeImage();
    this.setNavHeight();
  },

  resizeImage: function(){
    var header = document.getElementById('header'),
        footer = document.getElementById('footer'),
        main = document.getElementById('main-content'),
        vp = window.innerHeight,
        bodyHeight = document.body.clientHeight,
        hHeight = header.clientHeight,
        fHeight = footer.clientHeight,
        withMinHeight = hHeight + fHeight + 830;

    if(withMinHeight <  vp &&  bodyHeight < vp){
      var newHeight = (vp - (hHeight+fHeight)) + 'px';
      main.style.height = newHeight;
    }
  },

  makeNavSticky: function(){
    this.setNavHeight();

    $('.docs-sidebar').sticky({
      bottomSpacing: $('#footer').outerHeight(true)
    });
  },

  setNavHeight: function(){
    var viewportHeight = Math.max(document.documentElement.clientWidth, window.innerHeight || 0),
        maxHeight = Math.min(viewportHeight, $('#main-content').outerHeight());

    $('.docs-sidebar').css({
      height: maxHeight + 'px'
    });
  }

};

Init.start();

})();
