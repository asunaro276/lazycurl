## REMOVED Requirements

### Requirement: モード切り替え
**Reason**: `Adhoc`/`Collections`のモード概念そのものを廃止し、常時4パネル固定の単一画面に統合したため
**Migration**: `tui-shell`capabilityの「パネルベースのレイアウト」要求を参照

### Requirement: Adhocモードのレイアウト
**Reason**: モード別のレイアウトが存在しなくなり、常時4パネル固定レイアウトの一部になったため
**Migration**: `tui-shell`capabilityの「パネルベースのレイアウト」要求を参照

### Requirement: Adhocリクエストの一時性
**Reason**: 「Adhocモード」という区分ではなく「collectionに属していないRequestパネルの状態」として意味づけが変わったため
**Migration**: `tui-shell`capabilityの「collectionに属さないRequestの一時性」要求へ移管

### Requirement: collectionへの保存
**Reason**: モードに紐づかない`[0] Request`パネルの操作として`tui-shell`capabilityへ移管したため
**Migration**: `tui-shell`capabilityの「collectionへの保存」要求へ移管

### Requirement: 履歴の共有
**Reason**: Historyパネルが常時1つしか存在しなくなり、モード間で履歴を共有するという概念自体が成立しなくなったため(自明に満たされる)
**Migration**: `tui-shell`capabilityの「実行履歴の表示」要求を参照。追加の移行作業は不要
