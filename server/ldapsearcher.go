package server

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-ldap/ldap"
	autheV1 "k8s.io/api/authentication/v1"
	"k8s.io/klog/v2"
)

func ldapSearch(username, password string) (*autheV1.UserInfo, error) {
	ldapconn, err := ldap.DialURL(WebhookConfig.LDAPURL)
	if err != nil {
		klog.V(3).Info(err)
		return nil, err
	}
	defer ldapconn.Close()

	// Authenticate as LDAP admin user
	err = ldapconn.Bind(WebhookConfig.LDAPBaseDN, WebhookConfig.LDAPAuth)
	if err != nil {
		klog.V(3).Info(err)
		return nil, err
	}

	// Execute LDAP Search request
	result, err := ldapconn.Search(ldap.NewSearchRequest(
		WebhookConfig.LDAPBindDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		fmt.Sprintf("(&(objectClass=posixGroup)(memberUid=%s))", username), // Filter
		nil,
		nil,
	))

	if err != nil {
		klog.V(3).Info(err)
		return nil, err
	}

	userResult, err := ldapconn.Search(ldap.NewSearchRequest(
		WebhookConfig.LDAPBindDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		fmt.Sprintf("(&(objectClass=posixAccount)(uid=%s))", username), // Filter
		nil,
		nil,
	))

	if err != nil {
		klog.V(3).Info(err)
		return nil, err
	}

	if len(result.Entries) == 0 {
		klog.V(3).Info("User does not exist")
		return nil, errors.New("User does not exist")
	} else {
		// 验证用户名密码是否正确
		if err := ldapconn.Bind(userResult.Entries[0].DN, password); err != nil {
			e := fmt.Sprintf("Failed to auth. %s\n", err)
			klog.V(3).Info(e)
			return nil, errors.New(e)
		} else {
			klog.V(3).Info(fmt.Sprintf("User %s Authenticated successfuly!", username))
		}
		// 拼接为kubernetes authentication 的用户格式
		user := new(autheV1.UserInfo)
		for _, v := range result.Entries {
			attrubute := v.GetAttributeValue("objectClass")
			if strings.Contains(attrubute, "posixGroup") {
				user.Groups = append(user.Groups, v.GetAttributeValue("cn"))
			}
		}

		u := userResult.Entries[0].GetAttributeValue("uid")
		user.UID = u
		user.Username = u
		return user, nil
	}
}
