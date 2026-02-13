.PHONY:
scan:
	trufflehog filesystem . --fail
	trufflehog git file://. --only-verified
	checkov -d . --compact
tf-plan:
	./bash/tf-plan-setup.sh
tf-apply-auto:
	./bash/tf-apply-setup.sh --auto-approve true
tf-apply:
	./bash/tf-apply-setup.sh --auto-approve false
tf: 
	$(MAKE) tf-plan 
	$(MAKE) tf-apply-auto
clean:
	cd terraform && terraform destroy && cd ..