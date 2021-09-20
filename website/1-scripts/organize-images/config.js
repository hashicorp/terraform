/**
 * @typedef {Object} Config
 * @property {string} imgOut where images will go
 * @property {string} contentDir where docs content lives
 * @property {string} contentType what type of content the docs are
 */
const config = {
  // This is where images currently live...
  imgIn: "../../images/",
  // ...and this is the root folder of where they WILL live
  imgOut: "../../images/",
  contentDir: "../../docs/",
  contentType: "mdx",
}

module.exports = config