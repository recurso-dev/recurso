from http import HTTPStatus
from typing import Any
from urllib.parse import quote
from uuid import UUID

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.checkout_success_response_200 import CheckoutSuccessResponse200
from ...models.error import Error
from ...types import UNSET, Response, Unset


def _get_kwargs(
    id: UUID,
    *,
    payment_intent: str | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["payment_intent"] = payment_intent

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/checkout/{id}/success".format(
            id=quote(str(id), safe=""),
        ),
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> CheckoutSuccessResponse200 | Error | None:
    if response.status_code == 200:
        response_200 = CheckoutSuccessResponse200.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())

        return response_400

    if response.status_code == 404:
        response_404 = Error.from_dict(response.json())

        return response_404

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[CheckoutSuccessResponse200 | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    payment_intent: str | Unset = UNSET,
) -> Response[CheckoutSuccessResponse200 | Error]:
    """Verify a checkout payment and report settlement status

     Never settles on trust. When `payment_intent` is provided, the intent is verified directly with
    Stripe — it must have succeeded AND carry this invoice's id in its metadata — before the invoice is
    marked paid through the ledger path (idempotent with the payment webhook, which remains the
    authoritative backstop). A declined or abandoned intent reports `failed`; asynchronous methods (ACH,
    SEPA) report `processing` until the webhook settles them. Without `payment_intent`, the current
    invoice status is reported unchanged.

    Args:
        id (UUID):
        payment_intent (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CheckoutSuccessResponse200 | Error]
    """

    kwargs = _get_kwargs(
        id=id,
        payment_intent=payment_intent,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    payment_intent: str | Unset = UNSET,
) -> CheckoutSuccessResponse200 | Error | None:
    """Verify a checkout payment and report settlement status

     Never settles on trust. When `payment_intent` is provided, the intent is verified directly with
    Stripe — it must have succeeded AND carry this invoice's id in its metadata — before the invoice is
    marked paid through the ledger path (idempotent with the payment webhook, which remains the
    authoritative backstop). A declined or abandoned intent reports `failed`; asynchronous methods (ACH,
    SEPA) report `processing` until the webhook settles them. Without `payment_intent`, the current
    invoice status is reported unchanged.

    Args:
        id (UUID):
        payment_intent (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CheckoutSuccessResponse200 | Error
    """

    return sync_detailed(
        id=id,
        client=client,
        payment_intent=payment_intent,
    ).parsed


async def asyncio_detailed(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    payment_intent: str | Unset = UNSET,
) -> Response[CheckoutSuccessResponse200 | Error]:
    """Verify a checkout payment and report settlement status

     Never settles on trust. When `payment_intent` is provided, the intent is verified directly with
    Stripe — it must have succeeded AND carry this invoice's id in its metadata — before the invoice is
    marked paid through the ledger path (idempotent with the payment webhook, which remains the
    authoritative backstop). A declined or abandoned intent reports `failed`; asynchronous methods (ACH,
    SEPA) report `processing` until the webhook settles them. Without `payment_intent`, the current
    invoice status is reported unchanged.

    Args:
        id (UUID):
        payment_intent (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CheckoutSuccessResponse200 | Error]
    """

    kwargs = _get_kwargs(
        id=id,
        payment_intent=payment_intent,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    payment_intent: str | Unset = UNSET,
) -> CheckoutSuccessResponse200 | Error | None:
    """Verify a checkout payment and report settlement status

     Never settles on trust. When `payment_intent` is provided, the intent is verified directly with
    Stripe — it must have succeeded AND carry this invoice's id in its metadata — before the invoice is
    marked paid through the ledger path (idempotent with the payment webhook, which remains the
    authoritative backstop). A declined or abandoned intent reports `failed`; asynchronous methods (ACH,
    SEPA) report `processing` until the webhook settles them. Without `payment_intent`, the current
    invoice status is reported unchanged.

    Args:
        id (UUID):
        payment_intent (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CheckoutSuccessResponse200 | Error
    """

    return (
        await asyncio_detailed(
            id=id,
            client=client,
            payment_intent=payment_intent,
        )
    ).parsed
