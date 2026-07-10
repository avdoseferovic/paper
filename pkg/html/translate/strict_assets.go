package translate

import "errors"

func (tr *translator) reportAssetError(category, ref string, err error) {
	if tr == nil || !tr.strictAssets || err == nil {
		return
	}
	if errors.Is(err, ErrURLPolicyDenied) {
		return
	}
	tr.assetErrs = append(tr.assetErrs, NewAssetError(category, ref, err))
}

func (tr *translator) strictAssetError() error {
	if tr == nil || len(tr.assetErrs) == 0 {
		return nil
	}
	return errors.Join(tr.assetErrs...)
}
