var server = require("../server")

var chai = require('chai');

module.exports = {
	loadFunction: function(done) {
		var func = server.loadFunction('example/test', 'hello');
		chai.assert.isFunction(func, "Test relative path");
		done();
	}
}
