(function(){

var Init = {

  start: function(){
    var classname = this.hasClass(document.body, 'page-sub'),
      _this = this;

    if (classname) {
      _this.addEventListeners();
    }

    _this.makeNavSticky();
    _this.buildNavSelect();
    _this.initCollapsing();
    _this.initFilter();
  },

  hasClass: function (elem, className) {
    return new RegExp(' ' + className + ' ').test(' ' + elem.className + ' ');
  },

  addEventListeners: function(){
    var _this = this;

    // ignore filter submit
    $('.nav-filter')
      .on('submit', function(event){ event.preventDefault(); })
      .on('reset', function(event){ _this.clearFilter() });

    $('.nav-filter-input')
      .on('input', $.proxy(_this.onFilterInput, _this))
      .on('keyup', $.proxy(_this.onFilterKeyDown, _this));

    $(window)
      .on('resize', $.proxy(_this.onResize, _this));

    _this.resizeImage();
  },

  buildNavSelect: function(){
    var $sidebar = $('.docs-sidebar'),
        $formGroup = $('<div class="form-group docs-nav-mobile visible-xs visible-sm">').append('<label>Navigation</label>'),
        $select = $('<select class="form-control">'),
        $options = $([]);

    // kick off recursive search/build for nav <option>s
    $options = $options.add(this.buildNavOptions($sidebar.find('ul').first(), 0));

    $select
      .append($options)
      .appendTo($formGroup)
      .change(this.onNavSelectChange);

    $formGroup.insertBefore('#main-content');
  },

  clearFilter: function() {
    var $input = $('.nav-filter-input'),
      fakeEvent = {
        target: $input
      };

    $input.val('');
    this.onFilterInput(fakeEvent);
  },

  initCollapsing: function() {
    var _this = this,
      $activeEl = $('.active');

    $('.docs-sidenav ul')
      .removeClass('nav-visible')
      .parents('li')
        .addClass('has-sublist')
          .children('a')
            .click(this.onCollapseClick);

    _this.$superLists = $('.has-sublist');
    _this.$superLists.filter('.active.has-sublist').addClass('is-expanded');

    if ($activeEl.offset().top + 50 > $(window).height()) {
      $activeEl[0].scrollIntoView();
      window.scrollTo(0,0);
    }
  },

  initFilter: function() {
    var _this = this;

    _this.$filterResetButton = $('.nav-filter button[type="reset"]');

    _this.$filterableLinks = $('.docs-sidenav li a');
    _this.$filterableLinks.each(function(i, el) {
      var $el = $(el);
      $el.data('text', $el.text().trim());
    });

    $('nav-visible').removeClass('nav-visible');
  },

  onCollapseClick: function(event) {
    event.preventDefault();
    $(this).closest('.has-sublist').toggleClass('is-expanded');
  },

  onFilterInput: function(event) {
    var _this = this,
        $input = $(event.target),
        filterString = $input.val().trim().toLowerCase(),
        matchLength = filterString.length;

    // reset collapses
    _this.$superLists.show()
      .addClass('is-expanded');

    // collapse everything except active, if blank search
    if (filterString.length == 0) {
      _this.$filterResetButton.hide();
      _this.$superLists.show()
        .filter(':not(.active)')
          .removeClass('is-expanded');
    } else {
      _this.$filterResetButton.show();
    }

    // run search
    _this.$filterableLinks.each(function(i, el) {
      var $link = $(el),
          $item = $link.closest('li'),
          linkText = $link.data('text'),
          matchIndex = linkText.toLowerCase().indexOf(filterString),
          matchEnd = matchIndex + matchLength;

      if (matchIndex == -1) { // hide nonmatches
        if (!$item.hasClass('has-sublist')) { // exclude superLists from hide
          $item.hide();
        }
        $link.html(linkText); // reset text to have no highlight
      } else {
        $link.html( // highlight search term
          linkText.substr(0, matchIndex)
          + '<mark>' + linkText.substr(matchIndex, matchLength) + '</mark>'
          + linkText.substr(matchEnd)
        );
        $item.show();
      }
    });

    // make sure non-empty superlists are visible, empty ones hidden
    if (filterString.length > 0) {
      _this.$superLists.each(function(index, list) {
        var $list = $(list);
        if ($list.find('li').filter(':visible').length > 0) {
          $list.show().addClass('is-expanded');
        } else {
          $list.hide().removeClass('is-expanded');
        }
      });
    }
  },

  // clear filter on esc
  onFilterKeyDown: function(event) {
    if (event.which == 27) {
      this.clearFilter();
    }
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
