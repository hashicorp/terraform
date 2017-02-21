(function(){

var Init = {

  start: function(){
    var classname = this.hasClass(document.body, 'page-sub');

    if (classname) {
      this.addEventListeners();
    }

    this.makeNavSticky();
    this.buildNavSelect();
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

  buildNavSelect: function(){
    var $sidebar = $('.docs-sidebar'),
        $formGroup = $('<div class="form-group docs-nav-mobile">').append('<label>Navigation</label>'),
        $select = $('<select class="form-control visible-xs visible-sm">'),
        $options = $([]);

    // hide sidebar
    // TODO use a better selector? or write this into templates instead of js
    $sidebar.closest('.col-md-4').wrap($('<div class="hidden-xs hidden-sm"></div>'));

    // kick off recursive search/build for nav <option>s
    $options = $options.add(this.buildNavOptions($sidebar.find('ul').first(), 0));

    $select
      .append($options)
      .appendTo($formGroup)
      .change(this.onNavSelectChange);

    $formGroup.insertBefore('#main-content');
  },

  onNavSelectChange: function(event){
    location = this.value;
  },

  buildNavOptions: function($list, level){
    var _this = this,
        $options = $([]);

    // look for <a>s and new <ul> inside list item
    $list.children('li').each(function(index, item){
      var $item = $(item),
          $link = $item.children('a').first(),
          $sublist = $item.children('ul').first(),
          $option,
          space = '',
          i = 0;

      // add indentation to indicate current level
      for (; i<level; i++) {
        space += '&nbsp;&nbsp;&nbsp;&nbsp;';
      }

      $option = $('<option>',{
        value: $link.attr('href'),
        html: space + $link.text(),
        selected: $item.hasClass('active')
      });

      $options = $options.add($option);

      // if there's a sub <ul>, re-run this method one level deeper
      if ($sublist.length > 0) {
        $options = $options.add( _this.buildNavOptions($sublist, (level+1)) );
      }
    });

    return $options;
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
