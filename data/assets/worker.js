onmessage = (event) => {
  importScripts('/assets/highlightjs/highlight.min.js');
  const result = self.hljs.highlightAuto(event.data);
  postMessage(result.value);
};
