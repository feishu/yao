/**
 * 火山引擎即时通讯服务接口类型声明
 * 
 * 此文件为volcengine/service/im/process.go中通过process.RegisterGroup注册的方法提供类型声明
 * 所有方法都挂载在volc.im命名空间下
 */

declare namespace volc.im {
  /**
   * 注册用户到IM系统
   * 支持批量注册多个用户，通过Users数组传入用户信息
   * 
   * @param params 注册用户参数
   * @returns 注册结果
   * @see https://www.volcengine.com/docs/6348/1125993
   */
  function registerUsers(params: {
    /** 要注册的用户信息 */
    Users: Array<{
      /** 用户ID */
      UserId: number;
      /** 用户昵称，长度不能超过100字符 */
      NickName?: string;
      /** 用户标签，用于全员广播 */
      Tags?: string[];
      /** 头像URL，长度不能超过500字符 */
      Portrait?: string;
      /** 扩展字段 */
      Ext?: Record<string, string>;
    }>;
  }): RegisterUsersResponse;

  /**
   * 创建会话（单聊或群聊）
   * 可设置会话名称、类型、管理员等属性
   * 
   * @param params 创建会话参数
   * @returns 创建会话结果
   * @see https://www.volcengine.com/docs/6348/337013
   */
  function createConversation(params: {
    /** 会话名称（可选） */
    Name?: string;
    /** 会话类型：1-单聊，2-群聊，100-直播群 */
    ConversationType?: number;
    /** 群主用户ID */
    Owner?: number;
    /** 会话描述 */
    Description?: string;
    /** 会话头像URL */
    AvatarUrl?: string;
    /** 会话公告 */
    Notice?: string;
    /** 会话扩展字段 */
    Ext?: Record<string, string>;
    /** 单聊时另一个成员的用户ID */
    OtherUserId?: number;
    /** 幂等ID，防止重复创建 */
    IdempotentId?: string;
    /** 信箱类型，用于逻辑隔离，默认值为0 */
    InboxType?: number;
  }): CreateConversationResponse;

  /**
   * 修改会话信息
   * 可修改会话名称、描述等属性
   * 
   * @param params 修改会话参数
   * @returns 修改会话结果
   * @see https://www.volcengine.com/docs/6348/337115
   */
  function modifyConversation(params: {
    /** 会话短ID */
    ConversationShortId: number;
    /** 会话名称（可选） */
    Name?: string;
    /** 会话描述（可选） */
    Description?: string;
    /** 会话公告（可选） */
    Notice?: string;
    /** 会话头像URL（可选） */
    AvatarUrl?: string;
    /** 会话扩展字段（可选） */
    Ext?: Record<string, string>;
  }): ModifyConversationResponse;

  /**
   * 检查用户是否在指定会话中
   * 返回用户是否是会话成员的信息
   * 
   * @param params 检查用户参数
   * @returns 检查结果
   * @see https://www.volcengine.com/docs/6348/336996
   */
  function isUserInConversation(params: {
    /** 会话短ID */
    ConversationShortId: number;
    /** 用户ID */
    UserId: number;
    /** 参与者用户ID */
    ParticipantUserId?: number;
  }): IsUserInConversationResponse;

  /**
   * 发送消息
   * 支持发送文本、图片、视频等多种类型消息
   * 
   * @param params 发送消息参数
   * @returns 发送消息结果
   * @see https://www.volcengine.com/docs/6348/337135
   */
  function sendMessage(params: {
    /** 会话短ID */
    ConversationShortId: number;
    /** 发送者用户ID */
    SenderUserId: number;
    /** 消息内容 */
    Content: string;
    /** 
     * 消息类型
     * 10001：文本
     * 10003：图片
     * 10004：视频
     * 10005：文件
     * 10006：音频
     * 10012：自定义消息
     */
    MessageType?: number;
    /** 消息扩展字段 */
    Ext?: Record<string, string>;
    /** 消息@的用户ID列表 */
    MentionedUsers?: number[];
    /** 消息可见用户ID列表 */
    VisibleUsers?: number[];
    /** 消息不可见用户ID列表 */
    InvisibleUsers?: number[];
    /** 
     * 消息优先级
     * 0：低优先级
     * 1：普通优先级
     * 2：高优先级
     */
    Priority?: number;
    /** 客户端消息ID，用于幂等处理 */
    ClientMsgId?: string;
    /** 消息对应时间戳，单位为毫秒 */
    CreateTime?: number;
    /** 引用消息信息 */
    RefMsgInfo?: {
      /** 被引用的消息ID */
      ReferencedMessageId: number;
      /** 消息引用时展示的文本内容 */
      Hint: string;
    };
  }): SendMessageResponse;

  /**
   * 撤回消息
   * 允许用户撤回已发送的消息
   * 
   * @param params 撤回消息参数
   * @returns 撤回消息结果
   * @see https://www.volcengine.com/docs/6348/337141
   */
  function recallMessage(params: {
    /**
     * 会话短ID
     * @example 123456789
     */
    ConversationShortId: number;
    /** 
     * 消息ID 
     * @example 987654321
     */
    MessageId: number | string;
    /** 
     * 参与者用户ID（可选）
     * @example 10001
     */
    ParticipantUserId?: number;
  }): RecallMessageResponse;

  /**
   * 删除会话消息
   * 从会话中删除指定消息
   * 
   * @param params 删除消息参数
   * @returns 删除消息结果
   * @see https://www.volcengine.com/docs/6348/337140
   */
  function deleteConversationMessage(params: {
    /** 会话短ID */
    ConversationShortId: number;
    /** 消息ID */
    MessageId: number | string;
  }): DeleteConversationMessageResponse;

  /**
   * 获取会话消息列表
   * 根据会话ID获取会话中的消息列表
   * 
   * @param params 获取消息参数
   * @returns 消息列表结果
   * @see https://www.volcengine.com/docs/6348/337138
   */
  function getConversationMessages(params: {
    /** 会话短ID */
    ConversationShortId: number;
    /** 查询起始位置 */
    Cursor?: number;
    /** 查询条数 */
    Limit?: number;
    /** 
     * 查询方向
     * 0：正向查询
     * 1：反向查询
     * 默认值为0，直播群只能取1
     */
    Reverse?: number;
    /** 消息ID列表（可选） */
    MessageIds?: (number | string)[];
  }): GetConversationMessagesResponse;

  /**
   * 销毁会话
   * 删除指定会话，清理相关数据
   * 
   * @param params 销毁会话参数
   * @returns 销毁会话结果
   * @see https://www.volcengine.com/docs/6348/337036
   */
  function destroyConversation(params: {
    /** 会话短ID */
    ConversationShortId: number;
  }): DestroyConversationResponse;

  // 响应类型定义
  
  interface RegisterUsersResponse {
    /** 结果状态码 */
    Code: number;
    /** 结果消息 */
    Message: string;
    /** 响应数据 */
    Data?: {
      /** 注册失败的用户列表 */
      FailedUsers?: Array<{
        /** 用户ID */
        UserId: number;
        /** 失败原因 */
        Reason: string;
        /** 错误码 */
        Code: string;
        /** 错误信息 */
        Message: string;
      }>;
    };
  }

  interface CreateConversationResponse {
    /** 结果状态码 */
    Code: number;
    /** 结果消息 */
    Message: string;
    /** 响应数据 */
    Data?: {
      /** 会话短ID */
      ConversationShortId: number;
      /** 会话ID */
      ConversationId: string;
      /** 是否已存在 */
      Exist?: boolean;
      /** 会话信息 */
      ConversationInfo?: {
        /** 应用ID */
        AppId: number;
        /** 会话短ID */
        ConversationShortId: number;
        /** 会话ID */
        ConversationId: string;
        /** 会话类型: 1-单聊, 2-群聊, 100-直播群 */
        ConversationType: number;
        /** 会话创建时间 */
        CreateTime: number;
        /** 创建者用户ID */
        CreatorUserId: number;
        /** 信箱类型 */
        InboxType: number;
        /** 最后修改时间 */
        ModifyTime: number;
        /** 群主用户ID */
        OwnerUserId: number;
        /** 会话头像URL */
        AvatarUrl?: string;
        /** 会话描述 */
        Description?: string;
        /** 会话扩展字段 */
        Ext?: Record<string, string>;
        /** 会话成员数 */
        MemberCount?: number;
        /** 会话名称 */
        Name?: string;
        /** 会话公告 */
        Notice?: string;
        /** 直播群在线人数 */
        OnlineCount?: number;
        /** 单聊中另一个用户ID */
        OtherUserId?: number;
        /** 会话状态: 0-正常, 1-已解散 */
        Status?: number;
      };
    };
  }

  interface ModifyConversationResponse {
    /** 结果状态码 */
    Code: number;
    /** 结果消息 */
    Message: string;
    /** 响应数据 */
    Data?: Record<string, any>;
  }

  interface IsUserInConversationResponse {
    /** 结果状态码 */
    Code: number;
    /** 结果消息 */
    Message: string;
    /** 响应数据 */
    Data?: {
      /** 是否在会话中 */
      IsInConversation: boolean;
    };
  }

  interface SendMessageResponse {
    /** 结果状态码 */
    Code: number;
    /** 结果消息 */
    Message: string;
    /** 响应数据 */
    Data?: {
      /** 消息ID */
      MessageId: number | string;
      /** 服务端消息ID */
      ServerMessageId?: number;
    };
  }

  interface RecallMessageResponse {
    /** 结果状态码 */
    Code: number;
    /** 结果消息 */
    Message: string;
    /** 响应数据 */
    Data?: Record<string, any>;
  }

  interface DeleteConversationMessageResponse {
    /** 结果状态码 */
    Code: number;
    /** 结果消息 */
    Message: string;
    /** 响应数据 */
    Data?: Record<string, any>;
  }

  interface GetConversationMessagesResponse {
    /** 结果状态码 */
    Code: number;
    /** 结果消息 */
    Message: string;
    /** 响应数据 */
    Data?: {
      /** 是否还有更多消息 */
      HasMore: boolean;
      /** 下一页起始位置 */
      NewCursor: number;
      /** 消息列表 */
      Messages?: Array<{
        /** 应用ID */
        AppId: number;
        /** 消息内容 */
        Content: string;
        /** 会话短ID */
        ConversationShortId: number;
        /** 会话ID */
        ConversationId?: string;
        /** 会话类型 */
        ConversationType: number;
        /** 消息创建时间 */
        CreateTime: number;
        /** 消息ID */
        MessageId: number | string;
        /** 服务端消息ID */
        ServerMessageId?: number;
        /** 消息类型 */
        MessageType: number;
        /** 消息发送者ID */
        SenderUserId: number;
        /** 是否已撤回 */
        IsRecalled: boolean;
        /** 扩展字段 */
        Ext?: Record<string, string>;
        /** 被@的用户列表 */
        MentionedUsers?: number[];
        /** 可见的用户列表 */
        VisibleUsers?: number[];
        /** 不可见的用户列表 */
        InvisibleUsers?: number[];
        /** 引用消息 */
        RefMsgInfo?: {
          /** 被引用的消息ID */
          ReferencedMessageId: number;
          /** 会话短ID */
          ConversationShortId: number;
          /** 会话类型 */
          ConversationType: number;
          /** 消息创建时间 */
          CreateTime: number;
          /** 消息文本内容提示 */
          Hint: string;
          /** 消息类型 */
          MessageType: number;
          /** 消息发送者ID */
          SenderUserId: number;
        };
      }>;
    };
  }

  interface DestroyConversationResponse {
    /** 结果状态码 */
    Code: number;
    /** 结果消息 */
    Message: string;
    /** 响应数据 */
    Data?: Record<string, any>;
  }
} 