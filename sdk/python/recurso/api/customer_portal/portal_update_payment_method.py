from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.portal_update_payment_method_body import PortalUpdatePaymentMethodBody
from ...models.portal_update_payment_method_response_200 import PortalUpdatePaymentMethodResponse200
from ...types import Response


def _get_kwargs(
    *,
    body: PortalUpdatePaymentMethodBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/portal/api/payment-method",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | PortalUpdatePaymentMethodResponse200 | None:
    if response.status_code == 200:
        response_200 = PortalUpdatePaymentMethodResponse200.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())

        return response_400

    if response.status_code == 401:
        response_401 = Error.from_dict(response.json())

        return response_401

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Error | PortalUpdatePaymentMethodResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    body: PortalUpdatePaymentMethodBody,
) -> Response[Error | PortalUpdatePaymentMethodResponse200]:
    """Update the logged-in customer's payment method

     Updates the vaulted payment method for the authenticated portal customer. The card metadata
    (brand/last4/expiry) is produced by the client-side gateway tokenization step — no raw PAN is ever
    accepted. The customer is resolved from the portal session; it can never be supplied in the body.

    Args:
        body (PortalUpdatePaymentMethodBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PortalUpdatePaymentMethodResponse200]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient,
    body: PortalUpdatePaymentMethodBody,
) -> Error | PortalUpdatePaymentMethodResponse200 | None:
    """Update the logged-in customer's payment method

     Updates the vaulted payment method for the authenticated portal customer. The card metadata
    (brand/last4/expiry) is produced by the client-side gateway tokenization step — no raw PAN is ever
    accepted. The customer is resolved from the portal session; it can never be supplied in the body.

    Args:
        body (PortalUpdatePaymentMethodBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PortalUpdatePaymentMethodResponse200
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    body: PortalUpdatePaymentMethodBody,
) -> Response[Error | PortalUpdatePaymentMethodResponse200]:
    """Update the logged-in customer's payment method

     Updates the vaulted payment method for the authenticated portal customer. The card metadata
    (brand/last4/expiry) is produced by the client-side gateway tokenization step — no raw PAN is ever
    accepted. The customer is resolved from the portal session; it can never be supplied in the body.

    Args:
        body (PortalUpdatePaymentMethodBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PortalUpdatePaymentMethodResponse200]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient,
    body: PortalUpdatePaymentMethodBody,
) -> Error | PortalUpdatePaymentMethodResponse200 | None:
    """Update the logged-in customer's payment method

     Updates the vaulted payment method for the authenticated portal customer. The card metadata
    (brand/last4/expiry) is produced by the client-side gateway tokenization step — no raw PAN is ever
    accepted. The customer is resolved from the portal session; it can never be supplied in the body.

    Args:
        body (PortalUpdatePaymentMethodBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PortalUpdatePaymentMethodResponse200
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
