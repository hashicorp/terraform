/* jshint unused:false */
/* global console */
(function(Engine){ 'use strict';

Engine.Typewriter = function(element){
	this.element = element;
	this.content = this.element.textContent.split('');
	this.element.innerHTML = '';
	this.element.style.visibility = 'visible';
};

Engine.Typewriter.prototype = {

	running: false,

	letterInterval : 0.02,
	spaceInterval  : 0.4,

	charCount: -1,
	waitSpace: false,

	toDraw: '',

	start: function(){
		if (!this.content.length) {
			return this;
		}

		this._last = this.letterInterval;
		this.running = true;
	},

	update: function(engine){
		var newChar;

		if (!this.running) {
			return this;
		}

		this._last += engine.tick;

		if (this.waitSpace && this._last < this.spaceInterval) {
			return this;
		}

		if (!this.waitSpace && this._last < this.letterInterval){
			return this;
		}

		this._last = 0;
		newChar = this.content.shift();
		this.toDraw += newChar;

		if (newChar === ',') {
			this.waitSpace = true;
		} else {
			this.waitSpace = false;
		}

		this.element.innerHTML = this.toDraw + '<span class="cursor">_</span>';

		if (!this.content.length) {
			this.running = false;
		}

		return this;
	}

};

})(window.Engine);
