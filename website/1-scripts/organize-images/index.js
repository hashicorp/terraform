/**
 * This script moves images to
 */
const glob = require("glob")
const path = require("path")
const fs = require("fs-extra")
const findLCS = require("./utils/find-longest-common-substring")
const config = require("./config")

/**
 * Process script flags
 */
const ENV = process.argv.includes("--dev") ? "dev" : "prod"

/**
 * Process script config
 */

// Set a place for images to go
let imgOut =
  path.join(__dirname, config.imgOut) ||
  path.join(__dirname, "../public/img/docs/")

const contentDir = path.join(__dirname, config.contentDir)

let contentPaths = []
switch (config.contentType) {
  case "markdown":
    contentPaths = [
      ...glob.sync(`${contentDir}/**/*.md`),
      ...glob.sync(`${contentDir}/**/*.markdown`),
    ]
    break
  case "mdx":
    contentPaths = glob.sync(`${contentDir}/**/*.mdx`)
    break
  default:
    break
}

if (ENV === "dev") imgOut += "test/"

/**
 * Gather up content data
 */
const allPagesUsingImages = contentPaths
  .map((fullPath) => {
    let content = fs.readFileSync(fullPath, "utf8")

    // Check the page to see if it's using images
    const imgRegex = new RegExp(/(?<=\/img\/docs\/).*?(?=(\s|\)))/, "gm")

    // Add all the images it's using to an array
    const imagesUsed = [...content.matchAll(imgRegex)].flat()

    return {
      path: fullPath,
      imagesUsed,
    }
  })
  // Filter out any pages that don't use images
  .filter((page) => page.imagesUsed.length !== 0)

/**
 * Contains all used images, deduped via Set()
 */
const allImagesUsed = Array.from(
  new Set(
    allPagesUsingImages
      .map((page) => {
        return page.imagesUsed
      })
      .flat()
      // Filter out some frankly strange results like ")" and "]"
      .filter((imgPath) => imgPath.includes("."))
  )
)

/**
 * Move images into proper subdirectories,
 * renaming all paths in all docs files
 * to match
 */
for (let imageFilename of allImagesUsed) {
  // For each image, loop over allpagesusingimages...
  const pagesUsingImageAsPaths = allPagesUsingImages
    // ...filter out any pages that aren't using the image...
    .filter((page) => page.imagesUsed.find((i) => i === imageFilename))
    // ...then return an array of just the paths of these pages
    .map((filteredPage) => filteredPage.path)

  let finalImagePath = ""

  /**
   * If there's just one page using the image,
   * create the image's final path using that
   * page's path before the filename
   */
  if (pagesUsingImageAsPaths.length === 1) {
    // Get just the relevant part of the path that tells us the segment leading to the page.
    // This is key to recreating the structure of the docs within /img/docs/
    const pathToPage = pagesUsingImageAsPaths[0]
      .split("/")
      // Filter out the filename
      .filter((pathPart) => {
        switch (config.contentType) {
          case "markdown":
            return !pathPart.includes(".md") || !pathPart.includes(".markdown")
          case "mdx":
            return !pathPart.includes(".mdx")
        }
      })
      .join("/")
      // Remove the content dir from the path
      .split(contentDir)[1]

    finalImagePath = imgOut + pathToPage + `/${imageFilename}`
  } else if (pagesUsingImageAsPaths.length >= 1) {
    /**
     * Otherwise, find the longest common substring of the paths
     * to determine a common ancestor folder to put the image in
     */

    let commonPath = findLCS(pagesUsingImageAsPaths)

    // Normalize the common path to ensure future steps work
    if (!commonPath.endsWith("/")) commonPath = commonPath.concat("/")

    // Similar to pathToPage above
    const commonPathToDocs = commonPath.split(contentDir)[1]

    // Final image path is:
    // - the image output directory, plus
    // - the relevant part of the common path representing where the file is used, plus
    // - the image's filename
    finalImagePath = imgOut + commonPathToDocs + imageFilename
  }

  // Adjust all links to the image in the docs files to reflect the new path
  pagesUsingImageAsPaths.forEach((pagePath) =>
    adjustImageLinks(imageFilename, finalImagePath, pagePath)
  )

  if (ENV === "dev") {
    fs.copy(path.join(__dirname, config.imgIn + imageFilename), finalImagePath)
  } else {
    fs.move(path.join(__dirname, imgOut + imageFilename), finalImagePath)
  }
}

// Rewrite each docs file's image paths
// - For each of the page objects in allPagesUsingImages, do this:
// -- Match all image links
// -- Loop over the matches
// -- replace each match with the fullPath of its corresponding image file. can be done by searching for the image (there can only be one after all)

function adjustImageLinks(imageFilename, newImagePath, filePath) {
  let content = fs.readFileSync(filePath, "utf8")

  const linkRegex = new RegExp(`[(]/img/docs/${imageFilename}`, "gm")
  content = content.replace(linkRegex, `(/img/docs/${newImagePath}`)

  if (ENV === "dev") {
    fs.outputFile(
      path.join(__dirname, `./test-out/${filePath.split(contentDir)[1]}`),
      content
    )
  } else {
    fs.outputFile(filePath, content)
  }
}
