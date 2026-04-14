/**
 * 浏览器渲染能力类型声明
 *
 * 此文件为 utils/browser/process.go 中通过 process.RegisterGroup 注册的方法提供类型声明。
 * 所有方法都挂载在 utils.browser 命名空间下。
 */

declare namespace utils {
  namespace browser {
    interface RenderOptions {
      /**
       * 输出模式
       * - base64: 返回 Base64 字符串
       * - file: 写入文件并返回文件信息
       */
      output?: "base64" | "file";

      /** output=file 时必填。绝对路径写本地文件，非绝对路径写入 data 文件系统。 */
      filename?: string;

      /** HTML 中相对资源的基础地址 */
      base_url?: string;

      /** 超时时间，单位毫秒，默认 30000 */
      timeout?: number;

      /** 页面内容写入后额外等待时间，单位毫秒 */
      wait?: number;

      /** 视口宽度，默认 1280 */
      width?: number;

      /** 视口高度，默认 720 */
      height?: number;

      /** 缩放比例，默认 1 */
      scale?: number;

      /** PDF 是否横向输出 */
      landscape?: boolean;

      /** PDF 是否打印背景 */
      print_background?: boolean;

      /** PNG 是否捕获完整页面 */
      full_page?: boolean;
    }

    interface FileResult {
      filename: string;
      content_type: string;
      size: number;
    }

    /**
     * 将 HTML 渲染为 PDF
     *
     * @param html HTML 字符串
     * @param options 渲染选项
     * @returns output=base64 时返回 Base64 字符串，output=file 时返回文件信息
     */
    function pdf(html: string, options?: RenderOptions): string | FileResult;

    /**
     * 将 HTML 渲染为 PNG
     *
     * @param html HTML 字符串
     * @param options 渲染选项
     * @returns output=base64 时返回 Base64 字符串，output=file 时返回文件信息
     */
    function png(html: string, options?: RenderOptions): string | FileResult;
  }
}
