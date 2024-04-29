package finder

import (
	"testing"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

func TestFindDynamicLinksFromJS(t *testing.T) {
	l := launcher.New().
		Headless(true).
		Set("ignore-certificate-errors", "1").
		MustLaunch()
	b := rod.New().ControlURL(l).MustConnect()

	sourceCode := `!function(e){function n(n){for(var t,o,i=n[0],c=n[1],u=0,a=[];u<i.length;u++)o=i[u],r[o]&&a.push(r[o][0]),r[o]=0;for(t in c)Object.prototype.hasOwnProperty.call(c,t)&&(e[t]=c[t]);for(s&&s(n);a.length;)a.shift()()}var t={},r={0:0};function o(n){if(t[n])return t[n].exports;var r=t[n]={i:n,l:!1,exports:{}};return e[n].call(r.exports,r,r.exports,o),r.l=!0,r.exports}o.e=function(e){var n=[],t=r[e];if(0!==t)if(t)n.push(t[2]);else{var i=new Promise(function(n,o){t=r[e]=[n,o]});n.push(t[2]=i);var c,u=document.createElement("script");u.charset="utf-8",u.timeout=120,o.nc&&u.setAttribute("nonce",o.nc),u.src=function(e){return o.p+"chunks/"+({1:"todo"}[e]||e)+"."+{1:"d41d8cd98f00b204e980"}[e]+".js"}(e),c=function(n){u.onerror=u.onload=null,clearTimeout(s);var t=r[e];if(0!==t){if(t){var o=n&&("load"===n.type?"missing":n.type),i=n&&n.target&&n.target.src,c=new Error("Loading chunk "+e+" failed.\n("+o+": "+i+")");c.type=o,c.request=i,t[1](c)}r[e]=void 0}};var s=setTimeout(function(){c({type:"timeout",target:u})},12e4);u.onerror=u.onload=c,document.head.appendChild(u)}return Promise.all(n)},o.m=e,o.c=t,o.d=function(e,n,t){o.o(e,n)||Object.defineProperty(e,n,{enumerable:!0,get:t})},o.r=function(e){"undefined"!=typeof Symbol&&Symbol.toStringTag&&Object.defineProperty(e,Symbol.toStringTag,{value:"Module"}),Object.defineProperty(e,"__esModule",{value:!0})},o.t=function(e,n){if(1&n&&(e=o(e)),8&n)return e;if(4&n&&"object"==typeof e&&e&&e.__esModule)return e;var t=Object.create(null);if(o.r(t),Object.defineProperty(t,"default",{enumerable:!0,value:e}),2&n&&"string"!=typeof e)for(var r in e)o.d(t,r,function(n){return e[n]}.bind(null,r));return t},o.n=function(e){var n=e&&e.__esModule?function(){return e.default}:function(){return e};return o.d(n,"a",n),n},o.o=function(e,n){return Object.prototype.hasOwnProperty.call(e,n)},o.p="",o.oe=function(e){throw console.error(e),e};var i=window.webpackJsonp=window.webpackJsonp||[],c=i.push.bind(i);i.push=n,i=i.slice();for(var u=0;u<i.length;u++)n(i[u]);var s=c;o(o.s=0)}([function(e,n,t){"use strict";t.r(n);var r={title:"Main Application"},o={init:function(){this.appElement=document.querySelector("#app"),this.initEvents(),this.render()},initEvents:function(){var e=this;this.appElement.addEventListener("click",function(e){"btn-todo"===e.target.className&&t.e(1).then(t.bind(null,1)).then(function(e){e.TodoModule.init()}).catch(function(e){return"An error occurred while loading Module"})}),document.querySelector(".banner").addEventListener("click",function(n){n.preventDefault(),e.render()})},render:function(){this.appElement.innerHTML='\n    <section class="app">\n        <h3> '.concat(r.title,' </h3>\n        <section class="button">\n            <button class="btn-todo"> Todo Module </button>\n        </section>\n    </section>\n')}};({init:function(){this.initComponents(),this.initServiceWorker()},initComponents:function(){o.init()},initServiceWorker:function(){navigator.serviceWorker&&navigator.serviceWorker.register("./sw.js").then(function(){console.log("sw registered successfully!")}).catch(function(e){console.log("Some error while registering sw:",e)})}}).init()}]);`
	dynamicLinks := FindDynamicLinksFromJS(sourceCode, b)
	if len(dynamicLinks) != 1 {
		t.Errorf("len(dynamicLinks) should be 1, not %d", len(dynamicLinks))
	}
	if dynamicLinks[0] != "chunks/todo.d41d8cd98f00b204e980.js" {
		t.Error("wrong link")
	}
}

func TestFindLinksFromJS(t *testing.T) {
	sourceCode := `[{"name":"default","url":"/v2/api-docs","swaggerVersion":"2.0","location":"/v2/api-docs"}]`
	links := FindLinksFromJS(sourceCode)
	if len(links) != 1 {
		t.Errorf("len(links) should be 1, not %d", len(links))
	}
	if links[0] != "/v2/api-docs" {
		t.Error("wrong link")
	}
}
