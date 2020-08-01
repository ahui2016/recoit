# Notes about Recoit



## Reco 一条记录

- 一条 Reco 可能是一个 File(文件), 也可能是一个 Message(短消息).
- 如果 Reco.Type == "NotFile" 那么这就是一个 Message, 否则就是一个文件.



## Collection 集合

- 集合是很常用的，用户可能需要创建非常多的集合。
- 每个集合需要用户自定义一个标题，这会增加心智负担。【问题】
  - 自动用 Reco.Message 的开头二十个字符作为标题？
  - 如果没有 Message 则自动用第一个文件名作为标题？
  - 标题允许重复？
- 



## demo

- 只上传小图
- Collection.Title 限定长度